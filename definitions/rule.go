package definitions

// Dependency between Rules
type Dependency struct {
	Component string `yaml:"component"`
	Rule      string `yaml:"rule"`
	Export    string `yaml:"export"`
	Recurse   int    `yaml:"recurse"`
}

// Providers specifies the name of the Provider type to be used for the
// input and output Resources of a Rule
type Providers struct {
	Inputs  string `yaml:"inputs"`
	Outputs string `yaml:"outputs"`
}

// Rule defines inputs, commands, and outputs for a build step or action
type Rule struct {
	Name        string       `yaml:"name"`
	Inputs      []string     `yaml:"inputs"`
	Outputs     []string     `yaml:"outputs"`
	Ignore      []string     `yaml:"ignore"`
	Local       bool         `yaml:"local"`
	Native      bool         `yaml:"native"`
	Requires    []Dependency `yaml:"requires"`
	Description string       `yaml:"description"`
	Command     string       `yaml:"command"`
	Commands    []string     `yaml:"commands"`
	Providers   Providers    `yaml:"providers"`
}

func mergeRule(a, b Rule) Rule {
	return Rule{
		Inputs:      mergeStrings(a.Inputs, b.Inputs),
		Outputs:     mergeStrings(a.Outputs, b.Outputs),
		Ignore:      mergeStrings(a.Ignore, b.Ignore),
		Local:       mergeBool(a.Local, b.Local),
		Native:      mergeBool(a.Native, b.Native),
		Requires:    mergeDependencies(a.Requires, b.Requires),
		Description: mergeStr(a.Description, b.Description),
		Command:     mergeStr(a.Command, b.Command),
		Commands:    mergeStrings(a.Commands, b.Commands),
		Providers: Providers{
			Inputs:  mergeStr(a.Providers.Inputs, b.Providers.Inputs),
			Outputs: mergeStr(a.Providers.Outputs, b.Providers.Outputs),
		},
	}
}

func mergeRules(a, b map[string]Rule) map[string]Rule {

	names := map[string]bool{}
	for k := range a {
		names[k] = true
	}
	for k := range b {
		names[k] = true
	}

	result := map[string]Rule{}
	for name := range names {
		r := mergeRule(a[name], b[name])
		r.Name = name
		result[name] = r
	}
	return result
}

func mergeDependencies(a, b []Dependency) (result []Dependency) {
	if len(b) > 0 {
		for _, dep := range b {
			result = append(result, dep)
		}
		return
	}
	for _, dep := range a {
		result = append(result, dep)
	}
	return
}
