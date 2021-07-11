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
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/fugue/zim/definitions"
	"github.com/fugue/zim/envsub"
)

// Dependency on another Component (a Rule or an Export)
type Dependency struct {
	Component    string
	Rule         string
	Export       string
	Parameters   map[string]interface{}
	Recurse      int
	ResolvedRule *Rule
}

// Command to be run by a Rule
type Command struct {
	Kind       string
	Argument   string
	Attributes map[string]interface{}
}

// NewCommands constructs Commands extracted from a rule YAML definition
func NewCommands(self *definitions.Rule) (result []*Command, err error) {
	defCommands, err := self.GetCommands()
	if err != nil {
		return nil, err
	}
	// This form is used when the rule has a simple string for a command
	if len(defCommands) == 0 {
		result = []*Command{{Kind: "run", Argument: self.Command}}
		return
	}
	// Otherwise, the rule has a series of commands
	result = make([]*Command, 0, len(defCommands))
	for _, c := range defCommands {
		result = append(result, &Command{
			Kind:       c.Kind,
			Argument:   c.Argument,
			Attributes: c.Attributes,
		})
	}
	return
}

// Parameter is used to configure a Rule
type Parameter struct {
	Name        string
	Description string
	Type        string
	Default     interface{}
}

// Rule is an operation on a Component
type Rule struct {
	component         *Component
	name              string
	parameterizedName string
	local             bool
	native            bool
	inputs            []string
	ignore            []string
	requires          []*Dependency
	outputs           []string
	description       string
	commands          []*Command
	resolvedDeps      []*Rule
	resolvedImports   []*Export
	inProvider        Provider
	outProvider       Provider
	when              Condition
	unless            Condition
	parameters        map[string]interface{}
}

// NewRule constructs a Rule from a provided YAML definition
func NewRule(
	name string,
	parameterizedName string,
	parameters map[string]interface{},
	c *Component,
	self *definitions.Rule,
) (*Rule, error) {

	commands, err := NewCommands(self)
	if err != nil {
		return nil, fmt.Errorf("failed to create rule commands: %v", err)
	}

	r := &Rule{
		component:         c,
		name:              name,
		parameterizedName: parameterizedName,
		description:       self.Description,
		local:             self.Local,
		native:            self.Native,
		inputs:            self.Inputs,
		ignore:            self.Ignore,
		outputs:           self.Outputs,
		commands:          commands,
		requires:          make([]*Dependency, 0, len(self.Requires)),
		parameters:        parameters,
	}

	for _, dep := range self.Requires {
		ruleDep := &Dependency{
			Component:  dep.Component,
			Rule:       dep.Rule,
			Export:     dep.Export,
			Recurse:    dep.Recurse,
			Parameters: dep.Parameters,
		}
		if ruleDep.Component == "" {
			ruleDep.Component = c.Name()
		}
		r.requires = append(r.requires, ruleDep)
	}

	r.inProvider, err = c.Provider(self.Providers.Inputs)
	if err != nil {
		return nil, fmt.Errorf("rule %s provider error: %s", r.NodeID(), err)
	}
	r.outProvider, err = c.Provider(self.Providers.Outputs)
	if err != nil {
		return nil, fmt.Errorf("rule %s provider error: %s", r.NodeID(), err)
	}

	variables := envsub.GenericMap(r.BaseEnvironment())

	r.inputs, err = envsub.EvalStrings(r.inputs, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate inputs: %w", err)
	}
	r.outputs, err = envsub.EvalStrings(r.outputs, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate outputs: %w", err)
	}
	r.ignore, err = envsub.EvalStrings(r.ignore, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate ignore: %w", err)
	}

	r.when = Condition{
		ResourceExists:  self.When.ResourceExists,
		DirectoryExists: self.When.DirectoryExists,
		ScriptSucceeds: ConditionScript{
			Run:           self.When.ScriptSucceeds.Run,
			WithOutput:    self.When.ScriptSucceeds.WithOutput,
			SuppressError: self.When.ScriptSucceeds.SuppressError,
		},
	}
	r.unless = Condition{
		ResourceExists:  self.Unless.ResourceExists,
		DirectoryExists: self.Unless.DirectoryExists,
		ScriptSucceeds: ConditionScript{
			Run:           self.Unless.ScriptSucceeds.Run,
			WithOutput:    self.Unless.ScriptSucceeds.WithOutput,
			SuppressError: self.Unless.ScriptSucceeds.SuppressError,
		},
	}
	return r, nil
}

// resolveDeps accesses and stores dependencies of this Rule.
// This should be called internally after all components are loaded.
func (r *Rule) resolveDeps() error {
	for _, dep := range r.requires {
		// A Rule cannot depend on itself
		if r.Component().Name() == dep.Component && r.Name() == dep.Rule {
			return fmt.Errorf("invalid dependency - self reference: %s.%s",
				dep.Component, dep.Rule)
		}
		// If the dependency is an Export, this Rule is using source exported
		// from another Component
		if dep.Export != "" {
			export, err := r.resolveExport(dep)
			if err != nil {
				return err
			}
			r.resolvedImports = append(r.resolvedImports, export)
			continue
		}
		// Otherwise, this dependency is on the output of another Rule
		depRule, err := r.resolveDep(dep)
		if err != nil {
			return err
		}
		r.resolvedDeps = append(r.resolvedDeps, depRule)
	}
	return nil
}

// Accepts an export Dependency and returns the Export to which it refers.
func (r *Rule) resolveExport(dep *Dependency) (*Export, error) {
	if dep.Component == "" {
		return nil, fmt.Errorf("invalid dependency in %s: component name empty",
			r.NodeID())
	}
	if dep.Component == r.Component().Name() {
		return nil, fmt.Errorf("invalid dependency in %s: cannot import from self",
			r.NodeID())
	}
	export, err := r.Component().Project().Export(dep.Component, dep.Export)
	if err != nil {
		return nil, fmt.Errorf("invalid dependency in %s: %w", r.NodeID(), err)
	}
	return export, nil
}

// Accepts a Dependency and returns the Rule to which it refers.
// If the Dependency component name is blank, the component is assumed
// to be the one containing this Rule.
func (r *Rule) resolveDep(dep *Dependency) (*Rule, error) {
	depRule, err := r.Component().Project().Resolve(dep)
	if err != nil {
		return nil, err
	}
	dep.ResolvedRule = depRule
	return depRule, nil
}

// BaseEnvironment returns Rule environment variables that are known upfront
func (r *Rule) BaseEnvironment() map[string]string {
	env := r.Component().Environment() // this copy can be freely modified
	env["RULE"] = r.Name()
	env["NODE_ID"] = r.NodeID()
	for k, v := range r.parameters {
		env[k] = fmt.Sprintf("%v", v)
	}
	return env
}

// Environment returns variables to be used when executing this Rule
func (r *Rule) Environment() (map[string]string, error) {

	c := r.Component()
	var firstIn, firstOut, firstDep string

	// Inputs consumed by this Rule
	relInputs, err := r.Inputs()
	if err != nil {
		return nil, err
	}
	inputs, err := c.RelPaths(relInputs)
	if err != nil {
		return nil, err
	}
	if len(inputs) > 0 {
		firstIn = inputs[0]
	}

	// Outputs created by this Rule
	outputs, err := c.RelPaths(r.Outputs())
	if err != nil {
		return nil, err
	}
	if len(outputs) > 0 {
		firstOut = outputs[0]
	}

	// Dependencies consumed by this Rule
	relDeps, err := c.RelPaths(r.DependencyOutputs())
	if err != nil {
		return nil, err
	}
	if len(relDeps) > 0 {
		firstDep = relDeps[0]
	}

	tEnv := map[string]string{
		"INPUT":   firstIn,
		"OUTPUT":  firstOut,
		"OUTPUTS": strings.Join(outputs, " "),
		"DEP":     firstDep,
		"DEPS":    strings.Join(relDeps, " "),
	}
	return combineEnvironment(r.BaseEnvironment(), tEnv), nil
}

// Project containing this Rule
func (r *Rule) Project() *Project {
	return r.Component().Project()
}

// Component containing this Rule
func (r *Rule) Component() *Component {
	return r.component
}

// Name returns the rule name e.g. "build"
func (r *Rule) Name() string {
	return r.name
}

// NodeID makes Rules adhere to the graph.Node interface
func (r *Rule) NodeID() string {
	ruleName := r.parameterizedName
	if ruleName == "" {
		ruleName = r.name
	}
	return fmt.Sprintf("%s.%s", r.Component().Name(), ruleName)
}

// Image returns the Docker image used to build this Rule, if configured
func (r *Rule) Image() string {
	return r.Component().dockerImage
}

// IsNative returns true if Docker execution is disabled on this rule
func (r *Rule) IsNative() bool {
	return r.native || r.Image() == ""
}

// Dependencies of this rule. In order for this to Rule to run, its
// Dependencies should first be run.
func (r *Rule) Dependencies() []*Rule {
	return r.resolvedDeps
}

// HasOutputs returns true if this Rule produces one or more output Resources
func (r *Rule) HasOutputs() bool {
	return len(r.outputs) > 0
}

// Outputs returns Resources that are created by the Rule. The result here is
// NOT dependent on whether or not the Resources currently exist.
func (r *Rule) Outputs() (outputs Resources) {
	var prefix string
	switch r.outProvider.(type) {
	case *FileSystem:
		prefix = r.ArtifactsDir()
	}
	for _, out := range r.outputs {
		outputs = append(outputs, r.outProvider.New(path.Join(prefix, out)))
	}
	return
}

// ArtifactsDir returns the absolute path to the directory used for artifacts
// produced by this Rule.
func (r *Rule) ArtifactsDir() string {
	if r.local {
		return r.Component().Directory()
	}
	return r.Project().ArtifactsDir()
}

// MissingOutputs returns a list of output files that are not currently present
func (r *Rule) MissingOutputs() (missing Resources) {
	for _, out := range r.Outputs() {
		exists, _ := out.Exists()
		if !exists {
			missing = append(missing, out)
		}
	}
	return
}

// OutputsExist returns true if all rule output files are present on disk
func (r *Rule) OutputsExist() bool {
	return len(r.MissingOutputs()) == 0
}

// DependencyOutputs returns outputs of this Rule's dependencies
func (r *Rule) DependencyOutputs() (outputs Resources) {
	for _, dep := range r.Dependencies() {
		outputs = append(outputs, dep.Outputs()...)
	}
	return
}

// Commands that define Rule execution
func (r *Rule) Commands() []*Command {
	return r.commands
}

// Inputs returns Resources that are used to build this Rule
func (r *Rule) Inputs() (Resources, error) {

	addedPaths := map[string]bool{}
	ignoredPaths := map[string]bool{}
	var resources []Resource

	add := func(adding Resources) {
		for _, r := range adding {
			rPath := r.Path()
			if !addedPaths[rPath] {
				resources = append(resources, r)
				addedPaths[rPath] = true
			}
		}
	}

	ignore := func(removing Resources) {
		for _, r := range removing {
			ignoredPaths[r.Path()] = true
		}
	}

	// Find input resources
	inputs, err := matchResources(r.Component(), r.inProvider, r.inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to find input: %s", err)
	}
	add(inputs)

	// Exclude ignored resources
	ignored, err := matchResources(r.Component(), r.inProvider, r.ignore)
	if err != nil {
		return nil, fmt.Errorf("failed ignore: %s", err)
	}
	ignore(ignored)

	// Find resources imported from other Components
	for _, imp := range r.resolvedImports {
		imports, err := imp.Resolve()
		if err != nil {
			return nil, fmt.Errorf("failed to find import: %s", err)
		}
		add(imports)
	}

	// Return the input resources, less the ignored ones
	result := make(Resources, 0, len(resources))
	for _, r := range resources {
		if !ignoredPaths[r.Path()] {
			result = append(result, r)
		}
	}
	return result, nil
}

func matchResources(c *Component, p Provider, patterns []string) (result Resources, err error) {
	for _, pat := range patterns {
		matches, err := p.Match(path.Join(c.RelPath(), pat))
		if err != nil {
			return nil, err
		}
		result = append(result, matches...)
	}
	return
}

// RuleName returns the rule name for the given rule configuration
func RuleName(name string, parameters map[string]interface{}) string {
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
		result += fmt.Sprintf(" %s=%v", parameterName, parameters[parameterName])
	}
	return result
}
