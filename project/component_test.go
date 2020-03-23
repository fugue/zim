package project

import (
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fugue/zim/definitions"
)

func TestNewComponentError(t *testing.T) {
	self := &definitions.Component{}
	_, err := NewComponent(nil, self)
	if err == nil {
		t.Fatal("Empty definitions should error")
	}
	if err.Error() != "Component definition path is empty" {
		t.Fatal("Unexpected error:", err)
	}
}

func TestNewComponent(t *testing.T) {
	p := &Project{root: ".", rootAbs: "/repo"}
	self := &definitions.Component{
		Path: "/repo/foo/component.yaml",
	}
	c, err := NewComponent(p, self)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if c == nil {
		t.Fatal("Expected a Component to be returned; got nil")
	}
	if c.Directory() != "/repo/foo" {
		t.Error("Incorrect directory:", c.Directory())
	}
	if c.Name() != "foo" {
		t.Error("Incorrect name:", c.Name())
	}
	if len(c.Rules()) != 0 {
		t.Error("Expected empty rule names")
	}
	if c.HasRule("HUH?") {
		t.Error("Expected rule not to be found")
	}
	if _, ruleFound := c.Rule("WHUT"); ruleFound {
		t.Error("Expected rule not to be found")
	}
}

func TestNewComponentEmptyRule(t *testing.T) {
	p := &Project{
		root:      ".",
		rootAbs:   "/repo",
		providers: map[string]Provider{},
	}
	self := &definitions.Component{
		Path: "/repo/foo/component.yaml",
		Rules: map[string]definitions.Rule{
			"build": definitions.Rule{},
		},
	}
	c, _ := NewComponent(p, self)
	if c == nil {
		t.Fatal("Expected a Component to be returned; got nil")
	}
	if !c.HasRule("build") {
		t.Fatal("Expected build rule to exist")
	}
	rule, found := c.Rule("build")
	if !found {
		t.Fatal("Expected build rule to exist")
	}
	if len(rule.Outputs()) != 0 {
		t.Error("Expected empty slices")
	}
	inputs, err := rule.Inputs()
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 0 {
		t.Error("Expected empty inputs")
	}
}

func TestNewComponentRule(t *testing.T) {

	dir := testDir()
	defer os.RemoveAll(dir)
	cDir, cDefPath := testComponentDir(dir, "foo")
	testComponentFile(cDir, "go.mod", "")
	testComponentFile(cDir, "main.go", "")
	testComponentFile(cDir, "foo.go", "")
	testComponentFile(cDir, "exclude_me.go", "")

	p := &Project{
		root:      dir,
		rootAbs:   dir,
		artifacts: path.Join(dir, "artifacts"),
	}
	self := &definitions.Component{
		Path: cDefPath,
		Rules: map[string]definitions.Rule{
			"build": definitions.Rule{
				Description: "build it!",
				Inputs:      []string{"${NAME}.go", "*.go", "go.mod"},
				Ignore:      []string{"exclude_me.go"},
				Outputs:     []string{"foo"},
				Command:     "go build",
			},
		},
		Environment: map[string]string{"VOLUME": "11"},
	}
	c, _ := NewComponent(p, self)
	if c == nil {
		t.Fatal("Expected a Component to be returned; got nil")
	}

	rule, found := c.Rule("build")
	if !found {
		t.Fatal("Expected build rule to exist")
	}
	if !reflect.DeepEqual(c.Select([]string{"unknown", "build"}), []*Rule{rule}) {
		t.Error("Expected build rule to be selected")
	}
	if !reflect.DeepEqual(c.Rules(), []*Rule{rule}) {
		t.Error("Expected build rule to be present")
	}

	inputs, err := rule.Inputs()
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if !reflect.DeepEqual(inputs.Paths(), []string{
		path.Join(dir, "src/foo/foo.go"),
		path.Join(dir, "src/foo/main.go"),
		path.Join(dir, "src/foo/go.mod"),
	}) {
		t.Error("Incorrect inputs:", inputs.Paths())
	}

	if !reflect.DeepEqual(rule.Outputs().Paths(), []string{
		path.Join(dir, "artifacts/foo"),
	}) {
		t.Error("Incorrect artifacts:", rule.Outputs().Paths())
	}

	missing := rule.MissingOutputs()
	if !reflect.DeepEqual(missing.Paths(), []string{
		path.Join(dir, "artifacts/foo"),
	}) {
		t.Error("Incorrect missing artifacts:", missing.Paths())
	}

	if len(rule.Commands()) != 1 {
		t.Fatal("Expected one command")
	}
	cmd := rule.Commands()[0]
	if cmd != "go build" {
		t.Error("Incorrect command:", cmd)
	}

	env := rule.BaseEnvironment()
	if !reflect.DeepEqual(env, map[string]string{
		"COMPONENT": "foo",
		"NAME":      "foo",
		"NODE_ID":   "foo.build",
		"RULE":      "build",
		"KIND":      "",
		"VOLUME":    "11",
	}) {
		t.Fatal("Incorrect base environment:", env)
	}

	env, err = rule.Environment()
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if !reflect.DeepEqual(env, map[string]string{
		"COMPONENT": "foo",
		"NAME":      "foo",
		"NODE_ID":   "foo.build",
		"RULE":      "build",
		"KIND":      "",
		"DEP":       "",
		"DEPS":      "",
		"INPUT":     "foo.go",
		"INPUTS":    "foo.go main.go go.mod",
		"OUTPUT":    "../../artifacts/foo",
		"OUTPUTS":   "../../artifacts/foo",
		"VOLUME":    "11",
	}) {
		t.Fatal("Incorrect environment:", env)
	}
}

func TestComponentsFiltering(t *testing.T) {

	comps := Components{
		&Component{name: "a", kind: "go"},
		&Component{name: "b", kind: "python"},
		&Component{name: "c", kind: "go"},
	}

	goComps := comps.WithKind("go")
	if len(goComps) != 2 {
		t.Fatal("Expected two go components")
	}
	if goComps[0].Name() != "a" {
		t.Error("Expected a")
	}
	if goComps[1].Name() != "c" {
		t.Error("Expected c")
	}
	bComp := comps.WithName("b")
	if len(bComp) != 1 {
		t.Fatal("Expected to get b only")
	}
	if bComp[0].Name() != "b" {
		t.Error("Expected to get b, got:", bComp[0].Name())
	}
}

func TestComponentToolchain(t *testing.T) {

	dir := testDir()
	defer os.RemoveAll(dir)
	cDir, cDefPath := testComponentDir(dir, "foo")
	testComponentFile(cDir, "go.mod", "")
	testComponentFile(cDir, "main.go", "")
	testComponentFile(cDir, "foo.go", "")
	testComponentFile(cDir, "exclude_me.go", "")

	p := &Project{
		root:      dir,
		rootAbs:   dir,
		toolchain: map[string]string{},
		executor:  NewBashExecutor(),
	}
	self := &definitions.Component{
		Path: cDefPath,
		Toolchain: definitions.Toolchain{
			Items: []definitions.ToolchainItem{
				{Name: "go", Command: "go version"},
			},
		},
		Environment: map[string]string{"VOLUME": "11"},
	}
	c, _ := NewComponent(p, self)
	require.NotNil(t, c)

	m, err := c.Toolchain()
	require.Nil(t, err)
	require.Len(t, m, 1)

	value := m["go"]
	require.True(t, strings.HasPrefix(value, "go version go1."))
}
