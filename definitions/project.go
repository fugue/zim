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
	"io/ioutil"

	"github.com/go-yaml/yaml"
)

// Variable definition for creating a shell environment
type Variable struct {
	Definition string
	Script     string
}

// Environment contains a list of variables used to define a shell environment
type Environment struct {
	Variables []*Variable
}

// Project defines project configuration in YAML
type Project struct {
	Name        string                            `yaml:"name"`
	Environment interface{}                       `yaml:"environment"`
	Components  []string                          `yaml:"components"`
	Providers   map[string]map[string]interface{} `yaml:"providers"`
}

// LoadProject loads a definition from the given text
func LoadProject(text []byte) (*Project, error) {
	def := &Project{}
	if err := yaml.Unmarshal(text, def); err != nil {
		return nil, err
	}
	return def, nil
}

// LoadProjectFromPath loads a definition from the specified file
func LoadProjectFromPath(path string) (*Project, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadProject(data)
}

// GetEnvironment returns the Environment as defined at the Project level
func (p Project) GetEnvironment() (*Environment, error) {
	return GetEnvironment(p.Environment)
}

// GetEnvironment returns one Environment definition from its semi-structured
// YAML form
func GetEnvironment(obj interface{}) (*Environment, error) {

	var variables []*Variable

	switch e := obj.(type) {

	case []interface{}:
		for _, item := range e {
			if itemStr, ok := item.(string); ok {
				variables = append(variables, &Variable{Definition: itemStr})
			} else if itemMap, ok := item.(map[interface{}]interface{}); ok {
				if len(itemMap) != 1 {
					return nil, fmt.Errorf("invalid environment item: %v", itemMap)
				}
				key, value := getOneKeyValue(itemMap)
				keyStr, ok := key.(string)
				if !ok {
					return nil, fmt.Errorf("environment key must be a string: %v", key)
				}
				valueMap, ok := value.(map[interface{}]interface{})
				if !ok {
					return nil, fmt.Errorf("environment value is invalid: %v", value)
				}
				script, ok := valueMap["run"].(string)
				if !ok {
					return nil, fmt.Errorf("environment variable does not define a run statement: %v", keyStr)
				}
				variables = append(variables, &Variable{
					Definition: keyStr,
					Script:     script,
				})
			} else {
				return nil, fmt.Errorf("invalid environment definition: %v", e)
			}
		}

	case map[interface{}]interface{}:
		for k, v := range e {
			variables = append(variables, &Variable{Definition: fmt.Sprintf("%v=%v", k, v)})
		}

	default:
		return nil, fmt.Errorf("invalid environment definition: %v", e)
	}
	return &Environment{Variables: variables}, nil
}
