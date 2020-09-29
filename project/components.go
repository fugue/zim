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

// WithRule filters the Components to those with matching rule names
func (comps Components) WithRule(rule ...string) Components {
	var result Components
	for _, c := range comps {
		for _, r := range rule {
			if _, found := c.rules[r]; found {
				result = append(result, c)
				break
			}
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

// FilterNames returns a slice of component names minus the given names
func (comps Components) FilterNames(names []string) []string {
	namesMap := make(map[string]bool, len(names))
	for _, name := range names {
		namesMap[name] = true
	}

	var filteredNames []string
	for _, comp := range comps {
		name := comp.Name()
		if !namesMap[name] {
			filteredNames = append(filteredNames, name)
		}
	}

	return filteredNames
}

// FilterKinds returns a slice of component kinds minus the given kinds
func (comps Components) FilterKinds(kinds []string) []string {
	kindsMap := make(map[string]bool, len(kinds))
	for _, kind := range kinds {
		kindsMap[kind] = true
	}

	seenKinds := make(map[string]bool, len(comps))
	var filteredKinds []string
	for _, comp := range comps {
		kind := comp.Kind()
		if !seenKinds[kind] {
			seenKinds[kind] = true

			if !kindsMap[kind] {
				filteredKinds = append(filteredKinds, kind)
			}
		}
	}

	return filteredKinds
}

// FilterRules returns a slice of component rules minus the given rules
func (comps Components) FilterRules(rules []string) []string {
	rulesMap := make(map[string]bool, len(rules))
	for _, rule := range rules {
		rulesMap[rule] = true
	}

	seenRules := make(map[string]bool, len(comps))
	var filteredRules []string
	for _, comp := range comps {
		for _, rule := range comp.Rules() {
			ruleName := rule.Name()
			if !seenRules[ruleName] {
				seenRules[ruleName] = true

				if !rulesMap[ruleName] {
					filteredRules = append(filteredRules, ruleName)
				}
			}
		}
	}

	return filteredRules
}
