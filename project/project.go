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
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fugue/zim/definitions"
	"github.com/fugue/zim/exec"
	"github.com/hashicorp/go-multierror"
)

// Project is a collection of Components that can be built and deployed
type Project struct {
	sync.Mutex
	name            string
	root            string
	rootAbs         string
	artifacts       string
	cacheDir        string
	components      []*Component
	toolchain       map[string]string
	providers       map[string]Provider
	providerOptions map[string]map[string]interface{}
	executor        exec.Executor
}

// Opts defines options used when initializing a Project
type Opts struct {
	Root          string
	ProjectDef    *definitions.Project
	ComponentDefs []*definitions.Component
	Providers     []Provider
	Executor      exec.Executor
}

// New returns a Project that resides at the given root directory
func New(root string) (*Project, error) {
	projDef, componentDefs, err := Discover(root)
	if err != nil {
		return nil, fmt.Errorf("Failed to discover components: %s", err)
	}
	return NewWithOptions(Opts{
		Root:          root,
		ProjectDef:    projDef,
		ComponentDefs: componentDefs,
	})
}

// NewWithOptions returns a project based on the given options
func NewWithOptions(opts Opts) (*Project, error) {

	root := opts.Root
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("Failed to resolve path %s: %s", root, err)
	}

	// Create artifacts directory at the root level of the repository
	artifacts := path.Join(rootAbs, "artifacts")
	if err := os.MkdirAll(artifacts, 0755); err != nil {
		return nil, fmt.Errorf("Failed to artifacts dir %s: %s",
			artifacts, err)
	}

	var executor exec.Executor
	if opts.Executor != nil {
		executor = opts.Executor
	} else {
		executor = exec.NewBashExecutor()
	}

	p := &Project{
		root:            root,
		rootAbs:         rootAbs,
		artifacts:       artifacts,
		cacheDir:        exec.XDGCache(),
		toolchain:       map[string]string{},
		providers:       map[string]Provider{},
		providerOptions: map[string]map[string]interface{}{},
		executor:        executor,
	}

	if opts.ProjectDef != nil {
		p.name = opts.ProjectDef.Name
	}

	for _, provider := range opts.Providers {
		p.providers[provider.Name()] = provider
		if opts.ProjectDef != nil {
			opts, found := opts.ProjectDef.Providers[provider.Name()]
			if found {
				if err := provider.Init(opts); err != nil {
					return nil, err
				}
			}
		}
	}

	// Create components from their definitions
	for _, def := range opts.ComponentDefs {
		component, err := NewComponent(p, def)
		if err != nil {
			return nil, fmt.Errorf("Failed to load component %s: %s", def.Name, err)
		}
		p.components = append(p.components, component)
	}

	// Resolve dependencies between rules and return the project
	return p, p.resolveDeps()
}

// resolveDeps processes dependencies between Rules
func (p *Project) resolveDeps() error {
	var result *multierror.Error
	for _, c := range p.components {
		if err := c.resolveDeps(); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result.ErrorOrNil()
}

// Name of the project
func (p *Project) Name() string {
	return p.name
}

// Components returns all Components within the project
func (p *Project) Components() Components {
	return p.components
}

// Root directory of the project
func (p *Project) Root() string {
	return p.root
}

// RootAbsPath returns the absolute path to the root of the project
func (p *Project) RootAbsPath() string {
	return p.rootAbs
}

// AbsPaths returns absolute file paths given paths relative to the project root
func (p *Project) AbsPaths(paths []string) []string {
	var result []string
	for _, pth := range paths {
		result = append(result, path.Join(p.RootAbsPath(), pth))
	}
	return result
}

// ArtifactsDir returns the absolute path to the directory used for artifacts
func (p *Project) ArtifactsDir() string {
	return p.artifacts
}

// Select returns components with matching names or kind
func (p *Project) Select(names, kinds []string) (Components, error) {

	allComponents := p.Components()

	if len(names) == 0 && len(kinds) == 0 {
		return allComponents, nil
	}
	selectedByName := map[string]bool{}
	for _, name := range names {
		selectedByName[name] = true
	}
	selectedByKind := map[string]bool{}
	for _, kind := range kinds {
		selectedByKind[kind] = true
	}
	availableByName := map[string]bool{}
	for _, c := range allComponents {
		availableByName[c.Name()] = true
	}

	// Check that all the selected component names are valid
	for name := range selectedByName {
		if found := availableByName[name]; !found {
			return nil, fmt.Errorf("Unknown component: %s", name)
		}
	}
	// Filter the set of components to ones that were selected
	var selected Components
	for _, c := range allComponents {
		if selectedByName[c.Name()] || selectedByKind[c.Kind()] {
			selected = append(selected, c)
		}
	}
	return selected, nil
}

// Rule returns the specified Rule and a boolean indicating whether it was found
func (p *Project) Rule(component, ruleName string) (*Rule, bool) {
	rule := p.Components().WithName(component).Rule(ruleName).First()
	if rule != nil {
		return rule, true
	}
	return nil, false
}

// Export returns the specified Export and a boolean indicating whether it was found
func (p *Project) Export(component, exportName string) (*Export, bool) {
	c := p.Components().WithName(component).First()
	if c == nil {
		return nil, false
	}
	return c.Export(exportName)
}

// Toolchain returns information for the given component about the build tool
// versions used in the build. Components that use the same toolchain query
// will result in using the previously discovered values. This function
// accounts for whether the command executes within a Docker container.
func (p *Project) Toolchain(c *Component) (map[string]string, error) {

	p.Lock()
	defer p.Unlock()

	ctx := context.Background()
	res := map[string]string{}

	// Get an appropriate executor for the Component in terms of whether it is
	// Docker enabled. Use the Project executor by default, if it is compatible.
	var executor exec.Executor
	if c.dockerImage != "" {
		// Component is Docker-enabled
		if !p.executor.UsesDocker() {
			return nil, fmt.Errorf("Component %s is Docker-enabled but the executor is not Dockerized", c.Name())
		}
		executor = p.executor
	} else {
		// Component is not using Docker
		if p.executor.UsesDocker() {
			executor = exec.NewBashExecutor()
		} else {
			executor = p.executor
		}
	}

	usingDocker := executor.UsesDocker()
	toolchainKey := func(command string) string {
		if usingDocker {
			return fmt.Sprintf("%s.%s", c.dockerImage, command)
		}
		return command
	}

	for _, item := range c.toolchain.Items {
		key := toolchainKey(item.Command)
		value, found := p.toolchain[key]
		if found {
			res[item.Name] = value
			continue
		}
		buf := bytes.Buffer{}
		ignore := bytes.Buffer{}
		if err := executor.Execute(ctx, exec.ExecOpts{
			Image:   c.dockerImage,
			Command: item.Command,
			Stdout:  &buf,
			Cmdout:  &ignore,
		}); err != nil {
			return nil, err
		}
		value = strings.TrimSpace(buf.String())
		res[item.Name] = value
		p.toolchain[key] = value
	}
	return res, nil
}

// Provider returns the Provider with the given name, creating it if possible
func (p *Project) Provider(name string) (Provider, error) {

	if name == "" {
		name = "file"
	}

	p.Lock()
	defer p.Unlock()

	if p.providers == nil {
		p.providers = map[string]Provider{}
	}

	provider, found := p.providers[name]
	if found {
		return provider, nil
	}

	var err error
	switch name {
	case "file":
		provider, err = NewFileSystem(p.rootAbs)
	case "docker":
		provider, err = NewDocker()
	default:
		return nil, fmt.Errorf("Unknown provider: %s", name)
	}
	if err != nil {
		return nil, err
	}

	p.providers[name] = provider
	return provider, nil
}
