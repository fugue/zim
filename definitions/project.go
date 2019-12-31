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
