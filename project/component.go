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
package project

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"

	"github.com/fugue/zim/definitions"
	"github.com/fugue/zim/envsub"
)

// NewComponent initializes a Component from its YAML definition.
func NewComponent(p *Project, self *definitions.Component) (*Component, error) {

	if self == nil {
		return nil, errors.New("component definition is nil")
	}
	if self.Path == "" {
		return nil, errors.New("component definition path is empty")
	}
	absPath, err := filepath.Abs(self.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %s", err)
	}
	componentDir := filepath.Dir(absPath)
	name := self.Name
	if name == "" {
		name = filepath.Base(componentDir)
	}
	relPath, err := filepath.Rel(p.RootAbsPath(), componentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to determine relative path: %s", err)
	}

	c := Component{
		project:      p,
		componentDir: componentDir,
		relPath:      relPath,
		kind:         self.Kind,
		app:          self.App,
		dockerImage:  self.Docker.Image,
		name:         name,
		rules:        make(map[string]*Rule, len(self.Rules)),
		exports:      make(map[string]*Export, len(self.Exports)),
		env:          self.Environment,
		def:          self,
	}

	for _, item := range self.Toolchain.Items {
		c.toolchain.Items = append(c.toolchain.Items, ToolchainItem{
			Name:    item.Name,
			Command: item.Command,
		})
	}

	for name, export := range self.Exports {
		providerType := "file"
		if export.Provider != "" {
			providerType = export.Provider
		}
		provider, err := p.Provider(providerType)
		if err != nil {
			return nil, err
		}
		c.exports[name] = &Export{
			Component: &c,
			Provider:  provider,
			Resources: export.Resources,
			Ignore:    export.Ignore,
		}
	}

	return &c, nil
}

// ToolchainItem is one part of a Component Toolchain
type ToolchainItem struct {
	Name    string
	Command string
}

// Toolchain identifies dependencies this component has on build tools.
// This may be used to identify changes in toolchain that may necessitate
// a rebuild.
type Toolchain struct {
	Items []ToolchainItem
}

// Component to build and deploy in a repository
type Component struct {
	project      *Project
	componentDir string
	relPath      string
	name         string
	app          string
	kind         string
	dockerImage  string
	rules        map[string]*Rule
	exports      map[string]*Export
	env          map[string]string
	toolchain    Toolchain
	def          *definitions.Component
}

// Project returns the Project that contains this Component
func (c *Component) Project() *Project {
	return c.project
}

// Name of the Component which must be unique within the repository
func (c *Component) Name() string {
	return c.name
}

// App is the application name relating to this Component. This field is
// not leveraged by Zim in any particular way, but may be used to correlate
// this Component with a deployed application for example.
func (c *Component) App() string {
	return c.app
}

// Kind of the Component which determines its base settings
func (c *Component) Kind() string {
	return c.kind
}

// Directory returns the absolute path to the Component directory
func (c *Component) Directory() string {
	return c.componentDir
}

// RelPath returns the relative path to the Component within the repository
func (c *Component) RelPath() string {
	return c.relPath
}

// Rel returns the relative path from the Component to the given path
func (c *Component) Rel(p string) (string, error) {
	var absPath string
	if !filepath.IsAbs(p) {
		// Assume path is relative to Project root
		var err error
		absPath, err = filepath.Abs(path.Join(c.Project().RootAbsPath(), p))
		if err != nil {
			return "", fmt.Errorf("Component %s rel path %s failed: %s",
				c.Name(), p, err)
		}
	} else {
		absPath = p
	}
	return filepath.Rel(c.Directory(), absPath)
}

// RelPaths returns relative paths from the Component to the given paths
func (c *Component) RelPaths(rs Resources) ([]string, error) {
	return rs.RelativePaths(c.Directory())
}

// RuleName returns the rule name for the given rule configuration
func (c *Component) RuleName(name string, parameters map[string]interface{}) string {
	if len(parameters) == 0 {
		return name
	}
	parameterNames := make([]string, 0, len(parameters))
	for k := range parameters {
		parameterNames = append(parameterNames, k)
	}
	sort.Strings(parameterNames)
	result := name
	for _, parameterName := range parameterNames {
		result += fmt.Sprintf("%s=%v", parameterName, parameters[parameterName])
	}
	return result
}

// HasRule returns true if the Component has a Rule defined with the given name
func (c *Component) HasRule(name string) bool {
	_, found := c.def.Rules[name]
	return found
}

// Rule returns the Component rule with the given name if it exists
func (c *Component) Rule(
	name string,
	optParameters ...map[string]interface{},
) (*Rule, error) {

	ruleDef, found := c.def.Rules[name]
	if !found {
		return nil, fmt.Errorf("unknown rule: %s.%s", c.Name(), name)
	}
	var parameters map[string]interface{}
	if len(optParameters) > 0 {
		parameters = optParameters[0]
	}
	// Raise an error if unknown parameters were supplied
	for paramName := range parameters {
		if _, found := ruleDef.Parameters[paramName]; !found {
			return nil, fmt.Errorf("unknown parameter for %s.%s: %s",
				c.Name(), name, paramName)
		}
	}
	// Build up state available to variable substitution
	state := map[string]interface{}{
		"COMPONENT": c.Name(),
		"NAME":      c.Name(),
		"KIND":      c.Kind(),
		"RULE":      name,
	}
	for paramName, value := range parameters {
		state[paramName] = value
	}

	// Resolve the value for each parameter
	values := map[string]interface{}{}
	for pName, param := range ruleDef.Parameters {
		value, ok := parameters[pName]
		if ok {
			values[pName] = value
		} else if param.Default != nil {
			values[pName] = param.Default
		} else {
			return nil, fmt.Errorf("required parameter was not set for %s.%s: %s",
				c.Name(), name, pName)
		}
		if err := typeCheckParameter(param.Type, values[pName]); err != nil {
			return nil, fmt.Errorf("incorrect parameter type for %s.%s %s: %w",
				c.Name(), name, pName, err)
		}
		if strValue, ok := values[pName].(string); ok {
			subValue, err := envsub.EvalString(strValue, state)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve parameter %s in %s.%s: %w",
					pName, c.Name(), name, err)
			}
			values[pName] = subValue
		}
	}

	// Determine rule name including its parameters
	fullName := RuleName(name, values)
	rule, found := c.rules[fullName]
	if found {
		return rule, nil
	}
	rule, err := NewRule(name, fullName, values, c, &ruleDef)
	if err != nil {
		return nil, fmt.Errorf("failed to create rule instance: %w", err)
	}
	c.rules[fullName] = rule

	if err := rule.resolveDeps(); err != nil {
		return nil, err
	}
	return rule, nil
}

// MustRule returns the named rule or panics if it is not found
func (c *Component) MustRule(name string, parameters ...map[string]interface{}) *Rule {
	rule, err := c.Rule(name, parameters...)
	if err != nil {
		panic(err)
	}
	return rule
}

// RuleNames returns a list of all rule names defined in this Component
func (c *Component) RuleNames() []string {
	var names []string
	for name := range c.def.Rules {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Export returns the Component export with the given name, if it exists
func (c *Component) Export(name string) (*Export, error) {
	export, found := c.exports[name]
	if !found {
		return nil, fmt.Errorf("unknown export %s from component %s", name, c.Name())
	}
	return export, nil
}

// Exports returns a slice containing all Exports defined by this Component
func (c *Component) Exports() []*Export {
	exports := make([]*Export, 0, len(c.exports))
	for _, r := range c.exports {
		exports = append(exports, r)
	}
	return exports
}

// Environment returns environment variables applicable to this Component
func (c *Component) Environment() map[string]string {
	// Return a copy so that the original map cannot be modified
	env := copyEnvironment(c.env)
	env["COMPONENT"] = c.name
	env["NAME"] = c.name
	env["KIND"] = c.kind
	return env
}

// Toolchain returns this Components active toolchain information
func (c *Component) Toolchain() (map[string]string, error) {
	return c.Project().Toolchain(c)
}

// Provider returns the Provider with the given name
func (c *Component) Provider(name string) (Provider, error) {
	return c.Project().Provider(name)
}
