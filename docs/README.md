# go-simulator documentation

![Four simulators on the same made-up island: its relief, its climate, its vegetation after 200 years and a town after 100](img/chain.jpg)

*Four simulators taking turns on the same made-up island: the terrain gives the ground, the climate
works out heat, rain and biomes from it, the flora grows forests where they fit, and a town grows on
the plain. None of them knows the others: they only share the world.*

go-simulator is the engine those simulators share. It keeps the world, moves the clock forward and
passes each simulator what it needs. The simulators themselves are **plugins**: small separate
programs, in any language, that you can mix like building blocks.

## Where to start

| If you want to… | Read |
|---|---|
| understand how it all fits together | [Architecture](architecture.md) |
| set up a simulation | [Rules files](rules-files.md) |
| run it, with or without a window, and know what it writes | [Running](running.md) |
| write a simulator of your own | [Writing plugins](writing-plugins.md) |
| know which names plugins share, and the little condition language | [Vocabulary and conditions](vocabulary.md) |
| talk to the host from another language | [Protocol](protocol.md) |

Plugins that exist today: [terrain](https://github.com/akrck02/go-simulator-terrain), [climate](https://github.com/akrck02/go-simulator-climate),
[flora](https://github.com/akrck02/go-simulator-flora), [urban](https://github.com/akrck02/go-simulator-urban) and
[motion](https://github.com/akrck02/go-simulator-motion).
