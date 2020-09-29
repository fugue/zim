// Copyright 2020 Fugue, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package definitions

import (
	"fmt"
)

// Dependency between Rules
type Dependency struct {
	Component string `yaml:"component"`
	Rule      string `yaml:"rule"`
	Export    string `yaml:"export"`
	Recurse   int    `yaml:"recurse"`
}

// Providers specifies the name of the Provider type to be used for the
// input and output Resources of a Rule
type Providers struct {
	Inputs  string `yaml:"inputs"`
	Outputs string `yaml:"outputs"`
}

// Rule defines inputs, commands, and outputs for a build step or action
type Rule struct {
	Name        string        `yaml:"name"`
	Inputs      []string      `yaml:"inputs"`
	Outputs     []string      `yaml:"outputs"`
	Ignore      []string      `yaml:"ignore"`
	Local       bool          `yaml:"local"`
	Native      bool          `yaml:"native"`
	Requires    []Dependency  `yaml:"requires"`
	Description string        `yaml:"description"`
	Command     string        `yaml:"command"`
	Commands    []interface{} `yaml:"commands"`
	Providers   Providers     `yaml:"providers"`
}

// GetCommands returns commands unmarshaled from the rule's semi-structured YAML
func (r Rule) GetCommands() (result []*Command, err error) {
	result = make([]*Command, 0, len(r.Commands))
	for _, c := range r.Commands {
		command, err := GetCommand(c)
		if err != nil {
			return nil, err
		}
		result = append(result, command)
	}
	return
}

// GetCommand returns one command from its semi-structured YAML form
func GetCommand(obj interface{}) (*Command, error) {

	switch c := obj.(type) {

	// Simple form: a single string identifies the command kind
	case string:
		return &Command{Kind: c}, nil

	// Full form: a nested map defined the command
	case map[interface{}]interface{}:

		// There should be one key in the map that identifies the command kind
		if len(c) != 1 {
			return nil, fmt.Errorf("Invalid command schema: %+v", c)
		}
		key, value := getOneKeyValue(c)

		keyStr, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("Command key must be a string: %+v", c)
		}
		// This deals with commands structured like "run: echo hello"
		if valueStr, ok := value.(string); ok {
			return &Command{Kind: keyStr, Argument: valueStr}, nil
		}

		// This deals with commands containing attributes in a map
		valueMap, ok := value.(map[interface{}]interface{})
		if !ok {
			return nil, fmt.Errorf("Command values must be a map: %+v", c)
		}
		attributes, err := getMapWithStringKeys(valueMap)
		if err != nil {
			return nil, fmt.Errorf("Command attributes must have string keys: %+v", valueMap)
		}
		return &Command{Kind: keyStr, Attributes: attributes}, nil

	// Unrecognized command schema
	default:
		return nil, fmt.Errorf("Invalid command schema: %+v", obj)
	}
}

func getOneKeyValue(m map[interface{}]interface{}) (interface{}, interface{}) {
	for k, v := range m {
		return k, v
	}
	panic("Expected map to have a key")
}

func getMapWithStringKeys(m map[interface{}]interface{}) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	for k, v := range m {
		keyStr, ok := k.(string)
		if !ok {
			return nil, fmt.Errorf("Expected string key in map; got: %+v", k)
		}
		result[keyStr] = v
	}
	return result, nil
}

func mergeRule(a, b Rule) Rule {
	return Rule{
		Inputs:      mergeStrings(a.Inputs, b.Inputs),
		Outputs:     mergeStrings(a.Outputs, b.Outputs),
		Ignore:      mergeStrings(a.Ignore, b.Ignore),
		Local:       mergeBool(a.Local, b.Local),
		Native:      mergeBool(a.Native, b.Native),
		Requires:    mergeDependencies(a.Requires, b.Requires),
		Description: mergeStr(a.Description, b.Description),
		Command:     mergeStr(a.Command, b.Command),
		Commands:    mergeCommands(a.Commands, b.Commands),
		Providers: Providers{
			Inputs:  mergeStr(a.Providers.Inputs, b.Providers.Inputs),
			Outputs: mergeStr(a.Providers.Outputs, b.Providers.Outputs),
		},
	}
}

func mergeRules(a, b map[string]Rule) map[string]Rule {

	names := map[string]bool{}
	for k := range a {
		names[k] = true
	}
	for k := range b {
		names[k] = true
	}

	result := map[string]Rule{}
	for name := range names {
		r := mergeRule(a[name], b[name])
		r.Name = name
		result[name] = r
	}
	return result
}

func mergeDependencies(a, b []Dependency) (result []Dependency) {
	if len(b) > 0 {
		for _, dep := range b {
			result = append(result, dep)
		}
		return
	}
	for _, dep := range a {
		result = append(result, dep)
	}
	return
}

func mergeCommands(a, b []interface{}) (result []interface{}) {
	if len(b) > 0 {
		for _, cmd := range b {
			result = append(result, cmd)
		}
		return
	}
	for _, cmd := range a {
		result = append(result, cmd)
	}
	return
}
