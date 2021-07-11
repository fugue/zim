package definitions

// Parameter is used to configure a Rule
type Parameter struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Type        string      `yaml:"type"`
	Mode        string      `yaml:"mode"`
	Default     interface{} `yaml:"default"`
}

func mergeParameters(a, b map[string]Parameter) map[string]Parameter {
	result := map[string]Parameter{}
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		result[k] = v
	}
	return result
}
