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

// Docker defines Docker configuration for a component
type Docker struct {
	Image string `yaml:"image"`
}

// ECS defines ECS configuration for a component
type ECS struct {
	Task   string `yaml:"task"`
	Type   string `yaml:"type"`
	Memory int    `yaml:"memory"`
	CPU    int    `yaml:"cpu"`
}

// ToolchainItem is one part of a Component Toolchain
type ToolchainItem struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
}

// Toolchain identifies dependencies this component has on build tools.
// This may be used to identify changes in toolchain that may necessitate
// a rebuild.
type Toolchain struct {
	Items []ToolchainItem `yaml:"items"`
}

// Export defines resources exposed by a Component
type Export struct {
	Provider  string   `yaml:"provider"`
	Resources []string `yaml:"resources"`
	Ignore    []string `yaml:"ignore"`
}

// Component defines component configuration in YAML
type Component struct {
	Name        string            `yaml:"name"`
	App         string            `yaml:"app"`
	Kind        string            `yaml:"kind"`
	Ignore      bool              `yaml:"ignore"`
	Docker      Docker            `yaml:"docker"`
	ECS         ECS               `yaml:"ecs"`
	Toolchain   Toolchain         `yaml:"toolchain"`
	Rules       map[string]Rule   `yaml:"rules"`
	Exports     map[string]Export `yaml:"exports"`
	Environment map[string]string `yaml:"environment"`
	Path        string
}

// LoadComponent loads a definition from the given text
func LoadComponent(text []byte) (*Component, error) {
	def := &Component{}
	if err := yaml.Unmarshal(text, def); err != nil {
		return nil, err
	}
	return def, nil
}

// LoadComponentFromPath loads a definition from the specified file
func LoadComponentFromPath(path string) (*Component, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	def, err := LoadComponent(data)
	if err != nil {
		return nil, err
	}
	def.Path = path
	return def, nil
}

// Merge one Component defintion with another. Both original defintions remain
// unmodified and a new Component definition is returned.
func (c *Component) Merge(other *Component) *Component {
	r := &Component{
		Path:        other.Path,
		Name:        mergeStr(c.Name, other.Name),
		App:         mergeStr(c.App, other.App),
		Kind:        mergeStr(c.Kind, other.Kind),
		Ignore:      mergeBool(c.Ignore, other.Ignore),
		Docker:      mergeDocker(c.Docker, other.Docker),
		ECS:         mergeECS(c.ECS, other.ECS),
		Toolchain:   mergeToolchain(c.Toolchain, other.Toolchain),
		Rules:       mergeRules(c.Rules, other.Rules),
		Exports:     mergeExports(c.Exports, other.Exports),
		Environment: mergeStringsMap(c.Environment, other.Environment),
	}
	return r
}

func mergeStr(a, b string) string {
	if b != "" {
		return b
	}
	return a
}

func mergeInt(a, b int) int {
	if b != 0 {
		return b
	}
	return a
}

func mergeBool(a, b bool) bool {
	if b {
		return true
	}
	return a
}

func copyStrings(input []string) []string {
	if input == nil {
		return nil
	}
	result := make([]string, len(input))
	copy(result, input)
	return result
}

func mergeStrings(a, b []string) []string {
	if len(b) > 0 {
		return copyStrings(b)
	}
	return copyStrings(a)
}

func copyStringsMap(m map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range m {
		result[k] = v
	}
	return result
}

func copyExports(exports map[string]Export) map[string]Export {
	result := map[string]Export{}
	for k, export := range exports {
		result[k] = Export{
			Provider:  export.Provider,
			Resources: copyStrings(export.Resources),
			Ignore:    copyStrings(export.Ignore),
		}
	}
	return result
}

func mergeExports(a, b map[string]Export) map[string]Export {
	result := copyExports(a)
	for k, export := range b {
		result[k] = Export{
			Provider:  export.Provider,
			Resources: copyStrings(export.Resources),
			Ignore:    copyStrings(export.Ignore),
		}
	}
	return result
}

func mergeStringsMap(a, b map[string]string) map[string]string {
	result := copyStringsMap(a)
	for k, v := range b {
		result[k] = v
	}
	return result
}

func mergeDocker(a, b Docker) Docker {
	return Docker{Image: mergeStr(a.Image, b.Image)}
}

func mergeECS(a, b ECS) ECS {
	return ECS{
		CPU:    mergeInt(a.CPU, b.CPU),
		Memory: mergeInt(a.Memory, b.Memory),
		Task:   mergeStr(a.Task, b.Task),
		Type:   mergeStr(a.Type, b.Type),
	}
}

func mergeToolchain(a, b Toolchain) Toolchain {
	result := Toolchain{}
	for _, item := range a.Items {
		result.Items = append(result.Items, item)
	}
	if len(b.Items) > 0 {
		result.Items = nil
		for _, item := range b.Items {
			result.Items = append(result.Items, item)
		}
	}
	return result
}
