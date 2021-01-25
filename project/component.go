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
	"github.com/hashicorp/go-multierror"
)

// NewComponent initializes a Component from its YAML definition.
func NewComponent(p *Project, self *definitions.Component) (*Component, error) {

	if self == nil {
		return nil, errors.New("Component definition is nil")
	}
	if self.Path == "" {
		return nil, errors.New("Component definition path is empty")
	}
	absPath, err := filepath.Abs(self.Path)
	if err != nil {
		return nil, fmt.Errorf("Failed to resolve path: %s", err)
	}
	componentDir := filepath.Dir(absPath)
	name := self.Name
	if name == "" {
		name = filepath.Base(componentDir)
	}
	relPath, err := filepath.Rel(p.RootAbsPath(), componentDir)
	if err != nil {
		return nil, fmt.Errorf("Failed to determine relative path: %s", err)
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

	// for name, ruleDef := range self.Rules {
	// 	rule, err := NewRule(name, &c, &ruleDef)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	c.rules[name] = rule
	// }

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
	repoDir      string
	relPath      string
	artifactDir  string
	name         string
	app          string
	kind         string
	dockerImage  string
	sources      []string
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

// Rule returns the Component rule with the given name, if it exists,
// along with a boolean that indicates whether it was found
func (c *Component) Rule(name string, optParameters ...map[string]interface{}) (*Rule, bool) {

	// Retrieve the rule definition with that name
	ruleDef, found := c.def.Rules[name]
	fmt.Println("Rule", ruleDef, found)
	if !found {
		return nil, false
	}

	var parameters map[string]interface{}
	if len(optParameters) > 0 {
		parameters = optParameters[0]
	}

	// Resolve rule parameter values
	values := map[string]interface{}{}

	for pName, param := range ruleDef.Parameters {
		value, ok := parameters[pName]
		if ok {
			values[pName] = value
		} else {
			values[pName] = param.Default
		}
	}

	// Determine rule name which includes its parameters
	fullName := c.RuleName(name, values)
	rule, found := c.rules[fullName]
	if found {
		return rule, true
	}
	rule, err := NewRule(name, values, c, &ruleDef)
	if err != nil {
		return nil, false
	}

	if err := rule.resolveDeps(); err != nil {
		fmt.Println("resolveDeps ERROR", err)
		return nil, false
	}

	c.rules[fullName] = rule
	return rule, true
}

// MustRule returns the named rule or panics if it is not found
func (c *Component) MustRule(name string) *Rule {
	r, found := c.Rule(name)
	if !found {
		panic(fmt.Sprintf("Component %s has no rule %s", c.Name(), name))
	}
	return r
}

// Rules returns a slice containing all Rules defined by this Component
func (c *Component) Rules() []*Rule {
	rules := make([]*Rule, 0, len(c.rules))
	for _, r := range c.rules {
		rules = append(rules, r)
	}
	return rules
}

// HasRule returns true if a Rule with the given name is defined
func (c *Component) HasRule(name string) bool {
	_, found := c.rules[name]
	return found
}

// Export returns the Component export with the given name, if it exists
func (c *Component) Export(name string) (e *Export, found bool) {
	e, found = c.exports[name]
	return
}

// Exports returns a slice containing all Exports defined by this Component
func (c *Component) Exports() []*Export {
	exports := make([]*Export, 0, len(c.exports))
	for _, r := range c.exports {
		exports = append(exports, r)
	}
	return exports
}

// Select finds Rules belonging to this Component with the provided names.
// Unknown names are just ignored.
func (c *Component) Select(names []string) (result []*Rule) {
	for _, name := range names {
		if c.HasRule(name) {
			r, _ := c.Rule(name, nil)
			result = append(result, r)
		}
	}
	return
}

// Environment returns environment variables applicable to this Component
func (c *Component) Environment() map[string]string {
	// Return a copy so that the original map cannot be modified
	return copyEnvironment(c.env)
}

// resolveDeps processes inter-rule dependencies
func (c *Component) resolveDeps() error {
	var result *multierror.Error
	for _, rule := range c.rules {
		if err := rule.resolveDeps(); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result.ErrorOrNil()
}

// Toolchain returns this Components active toolchain information
func (c *Component) Toolchain() (map[string]string, error) {
	return c.Project().Toolchain(c)
}

// Provider returns the Provider with the given name
func (c *Component) Provider(name string) (Provider, error) {
	return c.Project().Provider(name)
}
