package protocol

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/akrck02/go-simulator/world"
)

// maximumHeaderBytes guards against reading garbage as a length (a plugin printing to stdout).
const maximumHeaderBytes = 64 << 20

// Message is a received message; Decode reads its body.
type Message struct {
	Kind        Kind
	Body        json.RawMessage
	Attachments [][]byte
}

type header struct {
	Kind        Kind            `json:"kind"`
	Body        json.RawMessage `json:"body,omitempty"`
	Attachments []int           `json:"attachments,omitempty"`
}

// Connection sends and receives frames over a pair of streams.
type Connection struct {
	reader  *bufio.Reader
	writer  *bufio.Writer
	sending sync.Mutex
}

// NewConnection talks over a reader (incoming) and a writer (outgoing).
func NewConnection(reader io.Reader, writer io.Writer) *Connection {
	return &Connection{reader: bufio.NewReaderSize(reader, 1<<16), writer: bufio.NewWriterSize(writer, 1<<16)}
}

// Send writes one message. Field values in the body travel as attachments.
func (connection *Connection) Send(kind Kind, body any) error {
	attachments := encodeFieldValues(fieldPatchesOf(body))
	encodedBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding %s: %w", kind, err)
	}
	encodedHeader, err := json.Marshal(header{Kind: kind, Body: encodedBody, Attachments: lengthsOf(attachments)})
	if err != nil {
		return err
	}
	connection.sending.Lock()
	defer connection.sending.Unlock()
	return connection.writeFrame(encodedHeader, attachments)
}

func (connection *Connection) writeFrame(encodedHeader []byte, attachments [][]byte) error {
	if err := binary.Write(connection.writer, binary.LittleEndian, uint32(len(encodedHeader))); err != nil {
		return err
	}
	if _, err := connection.writer.Write(encodedHeader); err != nil {
		return err
	}
	for _, attachment := range attachments {
		if _, err := connection.writer.Write(attachment); err != nil {
			return err
		}
	}
	return connection.writer.Flush()
}

// Receive reads one message.
func (connection *Connection) Receive() (Message, error) {
	var length uint32
	if err := binary.Read(connection.reader, binary.LittleEndian, &length); err != nil {
		return Message{}, err
	}
	if length > maximumHeaderBytes {
		return Message{}, fmt.Errorf("frame header of %d bytes: is the plugin writing to its standard output?", length)
	}
	encodedHeader := make([]byte, length)
	if _, err := io.ReadFull(connection.reader, encodedHeader); err != nil {
		return Message{}, err
	}
	var received header
	if err := json.Unmarshal(encodedHeader, &received); err != nil {
		return Message{}, fmt.Errorf("reading a frame header: %w", err)
	}
	attachments, err := connection.readAttachments(received.Attachments)
	return Message{Kind: received.Kind, Body: received.Body, Attachments: attachments}, err
}

func (connection *Connection) readAttachments(lengths []int) ([][]byte, error) {
	attachments := make([][]byte, 0, len(lengths))
	for _, length := range lengths {
		attachment := make([]byte, length)
		if _, err := io.ReadFull(connection.reader, attachment); err != nil {
			return nil, err
		}
		attachments = append(attachments, attachment)
	}
	return attachments, nil
}

// Decode reads the body into a value and gives field values back from the attachments.
func (message Message) Decode(body any) error {
	if len(message.Body) > 0 {
		if err := json.Unmarshal(message.Body, body); err != nil {
			return fmt.Errorf("reading %s: %w", message.Kind, err)
		}
	}
	return decodeFieldValues(fieldPatchesOf(body), message.Attachments)
}

// Expect checks the kind of a received message, turning a failure reply into an error.
func (message Message) Expect(kind Kind) error {
	if message.Kind == kind {
		return nil
	}
	if message.Kind == Failure {
		var failure FailureReply
		_ = message.Decode(&failure)
		return errors.New(failure.Message)
	}
	return fmt.Errorf("expected %s, received %s", kind, message.Kind)
}

// fieldPatchesOf lists the field patches a message body carries, in the order they travel.
func fieldPatchesOf(body any) []*world.FieldPatch {
	switch typed := body.(type) {
	case *world.ChangeSet:
		return patchesIn(typed.Changes)
	case world.ChangeSet:
		return patchesIn(typed.Changes)
	case *StartRequest:
		return patchesIn(typed.Snapshot.Changes)
	case StartRequest:
		return patchesIn(typed.Snapshot.Changes)
	case *StepRequest:
		return patchesIn(typed.Incoming)
	case StepRequest:
		return patchesIn(typed.Incoming)
	}
	return nil
}

func patchesIn(changes []world.Change) []*world.FieldPatch {
	var patches []*world.FieldPatch
	for _, change := range changes {
		if change.Operation == world.SetField && change.Field != nil {
			patches = append(patches, change.Field)
		}
	}
	return patches
}

func encodeFieldValues(patches []*world.FieldPatch) [][]byte {
	attachments := make([][]byte, 0, len(patches))
	for _, patch := range patches {
		bytes := make([]byte, 4*len(patch.Values))
		for index, value := range patch.Values {
			binary.LittleEndian.PutUint32(bytes[4*index:], math.Float32bits(value))
		}
		attachments = append(attachments, bytes)
	}
	return attachments
}

func decodeFieldValues(patches []*world.FieldPatch, attachments [][]byte) error {
	if len(patches) != len(attachments) {
		return fmt.Errorf("%d field changes but %d attachments", len(patches), len(attachments))
	}
	for index, patch := range patches {
		bytes := attachments[index]
		patch.Values = make([]float32, len(bytes)/4)
		for value := range patch.Values {
			patch.Values[value] = math.Float32frombits(binary.LittleEndian.Uint32(bytes[4*value:]))
		}
	}
	return nil
}

func lengthsOf(attachments [][]byte) []int {
	lengths := make([]int, 0, len(attachments))
	for _, attachment := range attachments {
		lengths = append(lengths, len(attachment))
	}
	return lengths
}
