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
	"io/ioutil"

	"github.com/go-yaml/yaml"
)

// Project defines project configuration in YAML
type Project struct {
	Name        string                            `yaml:"name"`
	Environment map[string]string                 `yaml:"environment"`
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
