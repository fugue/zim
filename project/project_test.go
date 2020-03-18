package project

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/LuminalHQ/zim/definitions"
)

func testDir(parent ...string) string {
	p := "/tmp"
	if len(parent) > 0 {
		p = parent[0]
	}
	dir, err := ioutil.TempDir(p, "zim-")
	if err != nil {
		panic(err)
	}
	return dir
}

func writeFile(name, text string) error {
	return ioutil.WriteFile(name, []byte(text), 0644)
}

func testComponent(root, name, def string, files map[string]string) {
	dir := path.Join(root, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}
	if err := writeFile(path.Join(dir, "component.yaml"), def); err != nil {
		panic(err)
	}
	for name, text := range files {
		if err := writeFile(path.Join(dir, name), text); err != nil {
			panic(err)
		}
	}
}

func testComponentDir(root, name string) (string, string) {
	cDir := path.Join(root, "src", name)
	if err := os.MkdirAll(cDir, 0755); err != nil {
		panic(err)
	}
	f, err := os.Create(path.Join(cDir, "component.yaml"))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf("name: %s", name))
	return cDir, path.Join(cDir, "component.yaml")
}

func testComponentFile(cDir, name, text string) string {
	f, err := os.Create(path.Join(cDir, name))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString(text)
	return name
}

func TestNewProject(t *testing.T) {

	dir := testDir()
	defer os.RemoveAll(dir)

	defs := []*definitions.Component{
		&definitions.Component{
			Name: "a",
			Kind: "python",
			Path: path.Join(dir, "a", "component.yaml"),
		},
		&definitions.Component{
			Name: "b",
			Kind: "go",
			Path: path.Join(dir, "b", "component.yaml"),
		},
	}

	projDef := &definitions.Project{Name: "example"}

	p, err := NewWithOptions(Opts{
		Root:          dir,
		ComponentDefs: defs,
		ProjectDef:    projDef,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.ArtifactsDir() != path.Join(dir, "artifacts") {
		t.Error("Incorrect artifacts directory")
	}
	components := p.Components()
	if len(components) != 2 {
		t.Fatal("Expected two components")
	}
	if components[0].Name() != "a" {
		t.Error("Expected 'a'")
	}
	if components[1].Name() != "b" {
		t.Error("Expected 'b'")
	}
	if p.Name() != "example" {
		t.Errorf("Incorrect project name: '%s'", p.Name())
	}
}

func TestSelectComponents(t *testing.T) {

	dir := testDir()
	defer os.RemoveAll(dir)

	defs := []*definitions.Component{
		&definitions.Component{
			Name: "a",
			Kind: "python",
			Path: path.Join(dir, "a", "component.yaml"),
		},
		&definitions.Component{
			Name: "b",
			Kind: "go",
			Path: path.Join(dir, "b", "component.yaml"),
		},
	}

	p, err := NewWithOptions(Opts{Root: dir, ComponentDefs: defs})
	if err != nil {
		t.Fatal(err)
	}
	selection, err := p.Select([]string{"a"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(selection) != 1 || selection[0].Name() != "a" {
		t.Errorf("Expected 'a'; got '%s'", selection[0].Name())
	}
	selection, err = p.Select(nil, []string{"go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(selection) != 1 || selection[0].Name() != "b" {
		t.Errorf("Expected 'b'; got '%s'", selection[0].Name())
	}
}

func TestNewProjectFromDisk(t *testing.T) {

	dir := testDir()
	defer os.RemoveAll(dir)

	testComponentDir(dir, "mycomp")

	p, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	comps := p.Components()
	if len(comps) != 1 {
		t.Fatal("Expected 1 component")
	}
	if comps[0].Name() != "mycomp" {
		t.Fatal("Expected mycomp")
	}
	paths := p.AbsPaths([]string{"src/foo"})
	if !reflect.DeepEqual(paths, []string{path.Join(dir, "src", "foo")}) {
		t.Error("Incorrect path:", paths)
	}
}

func TestResolveDeps(t *testing.T) {

	dir := testDir()
	defer os.RemoveAll(dir)

	defs := []*definitions.Component{
		&definitions.Component{
			Name: "a",
			Kind: "python",
			Path: path.Join(dir, "a", "component.yaml"),
			Rules: map[string]definitions.Rule{
				"build": definitions.Rule{
					Description: "a-build",
					Requires: []definitions.Dependency{
						definitions.Dependency{
							Component: "b",
							Rule:      "build",
						},
					},
				},
			},
		},
		&definitions.Component{
			Name: "b",
			Kind: "go",
			Path: path.Join(dir, "b", "component.yaml"),
			Rules: map[string]definitions.Rule{
				"build": definitions.Rule{
					Description: "b-build",
				},
			},
		},
	}

	p, err := NewWithOptions(Opts{Root: dir, ComponentDefs: defs})
	if err != nil {
		t.Fatal(err)
	}
	matches := p.Components().WithName("a")
	if len(matches) != 1 || matches[0].Name() != "a" {
		t.Fatal("Failed to find component 'a'")
	}
	a := matches[0]
	rule, found := a.Rule("build")
	if !found {
		t.Fatal("Failed to find build rule")
	}
	if rule.NodeID() != "a.build" {
		t.Errorf("Incorrect rule node ID: %s", rule.NodeID())
	}
	deps := rule.Dependencies()
	if len(deps) != 1 {
		t.Fatalf("Incorrect number of dependencies; got %d expected 1", len(deps))
	}
	dep := deps[0]
	if dep.NodeID() != "b.build" {
		t.Fatal("Expected dependency to be 'b.build'")
	}
}
