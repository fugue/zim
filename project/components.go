package project

// Components is a list of Components
type Components []*Component

// Rules is a list of Rules
type Rules []*Rule

// WithKind filters the Components to those with matching kind
func (comps Components) WithKind(kind ...string) Components {
	kinds := map[string]bool{}
	for _, k := range kind {
		kinds[k] = true
	}
	var result Components
	for _, c := range comps {
		if kinds[c.Kind()] {
			result = append(result, c)
		}
	}
	return result
}

// WithName filters the Components to those with matching names
func (comps Components) WithName(name ...string) Components {
	names := map[string]bool{}
	for _, n := range name {
		names[n] = true
	}
	var result Components
	for _, c := range comps {
		if names[c.Name()] {
			result = append(result, c)
		}
	}
	return result
}

// First component in the list, or nil if the list is empty
func (comps Components) First() *Component {
	if len(comps) > 0 {
		return comps[0]
	}
	return nil
}

// Rules returns a slice of all Rules with the given names across all
// these Components
func (comps Components) Rules(names []string) Rules {
	rules := make(Rules, 0, len(names)*len(comps))
	for _, c := range comps {
		for _, t := range c.Select(names) {
			rules = append(rules, t)
		}
	}
	return rules
}

// Rule returns a slice of all Rules with the given name across all
// these Components
func (comps Components) Rule(name string) Rules {
	rules := make(Rules, 0, len(comps))
	for _, c := range comps {
		if rule, found := c.Rule(name); found {
			rules = append(rules, rule)
		}
	}
	return rules
}

// First rule in the list, or nil if the list is empty
func (rules Rules) First() *Rule {
	if len(rules) > 0 {
		return rules[0]
	}
	return nil
}
