package project

import (
	"fmt"
	"path"
	"strings"

	"github.com/fugue/zim/definitions"
)

// Dependency on another Component (a Rule or an Export)
type Dependency struct {
	Component string
	Rule      string
	Export    string
	Recurse   int
}

// Rule is an operation on a Component
type Rule struct {
	component       *Component
	name            string
	local           bool
	native          bool
	inputs          []string
	ignore          []string
	requires        []*Dependency
	outputs         []string
	description     string
	commands        []string
	resolvedDeps    []*Rule
	resolvedImports []*Export
	inProvider      Provider
	outProvider     Provider
}

// NewRule constructs a Rule from a provided YAML definition
func NewRule(name string, c *Component, self *definitions.Rule) (*Rule, error) {

	r := &Rule{
		component:   c,
		name:        name,
		description: self.Description,
		local:       self.Local,
		native:      self.Native,
		inputs:      self.Inputs,
		ignore:      self.Ignore,
		outputs:     self.Outputs,
		commands:    self.Commands,
		requires:    make([]*Dependency, 0, len(self.Requires)),
	}

	for _, dep := range self.Requires {
		r.requires = append(r.requires, &Dependency{
			Component: dep.Component,
			Rule:      dep.Rule,
			Export:    dep.Export,
			Recurse:   dep.Recurse,
		})
	}

	if len(r.commands) == 0 && len(self.Command) > 0 {
		r.commands = []string{self.Command}
	}

	var err error
	r.inProvider, err = c.Provider(self.Providers.Inputs)
	if err != nil {
		return nil, fmt.Errorf("Rule %s provider error: %s", r.NodeID(), err)
	}
	r.outProvider, err = c.Provider(self.Providers.Outputs)
	if err != nil {
		return nil, fmt.Errorf("Rule %s provider error: %s", r.NodeID(), err)
	}

	variables := r.BaseEnvironment()
	r.inputs = substituteVarsSlice(r.inputs, variables)
	r.ignore = substituteVarsSlice(r.ignore, variables)
	r.outputs = substituteVarsSlice(r.outputs, variables)

	return r, nil
}

// resolveDeps accesses and stores dependencies of this Rule.
// This should be called internally after all components are loaded.
func (r *Rule) resolveDeps() error {
	for _, dep := range r.requires {
		// A Rule cannot depend on itself
		if r.Component().Name() == dep.Component && r.Name() == dep.Rule {
			return fmt.Errorf("Invalid dep - self reference: %s.%s",
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
		// Currently it is allowed to pull in transitive dependencies that
		// are one step removed as dependencies of this Rule, if desired.
		// This can be helpful when the immediate dependency doesn't actually
		// fully encapsulate its own dependencies outputs.
		if dep.Recurse > 1 {
			return fmt.Errorf("Invalid dep - recursion: %s.%s",
				dep.Component, dep.Rule)
		} else if dep.Recurse == 1 {
			// Pull in transitive dependencies that are one step removed
			for _, rDep := range depRule.requires {
				rDepRule, err := r.resolveDep(rDep)
				if err != nil {
					return err
				}
				r.resolvedDeps = append(r.resolvedDeps, rDepRule)
			}
		}
	}
	return nil
}

// Accepts an export Dependency and returns the Export to which it refers.
func (r *Rule) resolveExport(dep *Dependency) (*Export, error) {
	if dep.Component == "" {
		return nil, fmt.Errorf("Invalid dep in %s - component name empty",
			r.NodeID())
	}
	if dep.Component == r.Component().Name() {
		return nil, fmt.Errorf("Invalid dep in %s - cannot import from self",
			r.NodeID())
	}
	export, found := r.Component().Project().Export(dep.Component, dep.Export)
	if !found {
		return nil, fmt.Errorf("Invalid dep in %s - export not found: %s.%s",
			r.NodeID(), dep.Component, dep.Export)
	}
	return export, nil
}

// Accepts a Dependency and returns the Rule to which it refers.
// If the Dependency component name is blank, the component is assumed
// to be the one containing this Rule.
func (r *Rule) resolveDep(dep *Dependency) (*Rule, error) {

	var depCompName string
	if dep.Component == "" {
		depCompName = r.Component().Name()
	} else {
		depCompName = dep.Component
	}

	depRule, found := r.Component().Project().Rule(depCompName, dep.Rule)
	if !found {
		return nil, fmt.Errorf("Invalid dep - rule not found: %s.%s",
			depCompName, dep.Rule)
	}
	return depRule, nil
}

// BaseEnvironment returns Rule environment variables that are known upfront
func (r *Rule) BaseEnvironment() map[string]string {
	c := r.Component()
	return combineEnvironment(c.Environment(), map[string]string{
		"COMPONENT": c.Name(),
		"NAME":      c.Name(),
		"KIND":      c.Kind(),
		"RULE":      r.Name(),
		"NODE_ID":   r.NodeID(),
	})
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
	combined := combineEnvironment(r.BaseEnvironment(), tEnv)
	return combined, nil
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
	return fmt.Sprintf("%s.%s", r.Component().Name(), r.Name())
}

// Image returns the Docker image used to build this Rule, if configured
func (r *Rule) Image() string {
	return r.Component().dockerImage
}

// IsNative returns true if Docker execution is disabled on this rule
func (r *Rule) IsNative() bool {
	return r.native
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
		if r.local {
			prefix = r.Component().Directory()
		} else {
			prefix = r.Project().ArtifactsDir()
		}
	}
	for _, out := range r.outputs {
		outputs = append(outputs, r.outProvider.New(path.Join(prefix, out)))
	}
	return
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
		for _, out := range dep.Outputs() {
			outputs = append(outputs, out)
		}
	}
	return
}

// Commands that define Rule execution
func (r *Rule) Commands() []string {
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
		return nil, fmt.Errorf("Failed to find input: %s", err)
	}
	add(inputs)

	// Exclude ignored resources
	ignored, err := matchResources(r.Component(), r.inProvider, r.ignore)
	if err != nil {
		return nil, fmt.Errorf("Failed ignore: %s", err)
	}
	ignore(ignored)

	// Find resources imported from other Components
	for _, imp := range r.resolvedImports {
		imports, err := matchResources(imp.Component, imp.Provider, imp.Resources)
		if err != nil {
			return nil, fmt.Errorf("Failed to find import: %s", err)
		}
		add(imports)

		ignored, err := matchResources(imp.Component, imp.Provider, imp.Ignore)
		if err != nil {
			return nil, fmt.Errorf("Failed to ignore: %s", err)
		}
		ignore(ignored)
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
