package definitions

import (
	"io/ioutil"

	"github.com/go-yaml/yaml"
)

// Command defines one action to execute within a rule
type Command struct {
	Name        string            `yaml:"name"`
	Kind        string            `yaml:"kind"`
	Command     string            `yaml:"command"`
	Environment map[string]string `yaml:"environment"`
	Attributes  map[string]interface{}
	Path        string
}

// LoadCommand loads a definition from the given text
func LoadCommand(text []byte) (*Command, error) {
	def := &Command{}
	if err := yaml.Unmarshal(text, def); err != nil {
		return nil, err
	}
	return def, nil
}

// LoadCommandFromPath loads a definition from the specified file
func LoadCommandFromPath(path string) (*Command, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	def, err := LoadCommand(data)
	if err != nil {
		return nil, err
	}
	def.Path = path
	return def, nil
}
