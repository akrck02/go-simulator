package configuration

import (
	"fmt"
	"maps"
	"os"

	"gopkg.in/yaml.v3"
)

// includeKey is the parameter that brings in parameters from other files:
//
//   - rule: city
//     with: {include: town_scenario.yaml, seed: 8}
//
// The files are read in order (one path or a list), each one's parameters replacing the previous
// ones of the same name, and the rule's own parameters replace them all. The files are plain YAML
// maps, so a scenario can live in its own file, with comments, next to the notes it comes from.
// Paths inside them still start at the rules file's folder. A file cannot include others.
const includeKey = "include"

// expandIncludes replaces the include parameter of every rule by the parameters of its files.
func (configuration *Configuration) expandIncludes() error {
	for stageIndex := range configuration.Stages {
		stage := &configuration.Stages[stageIndex]
		for ruleIndex := range stage.Rules {
			rule := &stage.Rules[ruleIndex]
			expanded, err := configuration.withIncludes(rule.With)
			if err != nil {
				return fmt.Errorf("stage %q, rule %q: %w", stage.Name, rule.Rule, err)
			}
			rule.With = expanded
		}
	}
	return nil
}

func (configuration *Configuration) withIncludes(parameters map[string]any) (map[string]any, error) {
	written, present := parameters[includeKey]
	if !present {
		return parameters, nil
	}
	paths, err := includePaths(written)
	if err != nil {
		return nil, err
	}
	expanded := map[string]any{}
	for _, path := range paths {
		included, err := readIncluded(configuration.Resolve(path))
		if err != nil {
			return nil, err
		}
		maps.Copy(expanded, included)
	}
	maps.Copy(expanded, parameters)
	delete(expanded, includeKey)
	return expanded, nil
}

func includePaths(written any) ([]string, error) {
	if path, isText := written.(string); isText {
		return []string{path}, nil
	}
	list, isList := written.([]any)
	if !isList {
		return nil, fmt.Errorf("%q is a path or a list of paths", includeKey)
	}
	paths := make([]string, 0, len(list))
	for _, item := range list {
		path, isText := item.(string)
		if !isText {
			return nil, fmt.Errorf("%q is a path or a list of paths", includeKey)
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func readIncluded(path string) (map[string]any, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot include: %w", err)
	}
	var parameters map[string]any
	if err := yaml.Unmarshal(contents, &parameters); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if _, nested := parameters[includeKey]; nested {
		return nil, fmt.Errorf("%s: an included file cannot include others", path)
	}
	return parameters, nil
}
