package envsub

import (
	"fmt"
	"strings"

	"github.com/drone/envsubst"
)

// Eval performs variable substitution on parameters that may reference each
// other, and stores the final resolved key-values in the provided state map.
// The state map may already contain resolved key-values, and these may also
// be referenced by the parameters. Variable references must follow the envsubt
// format, e.g. ${var}.
func Eval(
	state map[string]interface{},
	parameters map[string]interface{},
) (finalErr error) {

	// Used to prevent infinite recursion
	seen := map[string]bool{}

	// Declaration to support recursion
	var eval func(s string) string

	// Evaluate a string, returning the final value with all variables resolved.
	// Calls itself recursively when there are chains of variable references.
	eval = func(s string) string {
		result, err := envsubst.Eval(s, func(key string) string {
			if stateValue, found := state[key]; found {
				return fmt.Sprintf("%v", stateValue)
			}
			// Return early if recursion is detected
			if seen[key] {
				finalErr = fmt.Errorf("recursion detected")
				return ""
			}
			seen[key] = true

			paramValue, found := parameters[key]
			if !found {
				finalErr = fmt.Errorf("unknown variable: %s", key)
				return ""
			}
			resolvedValue := eval(fmt.Sprintf("%v", paramValue))
			state[key] = resolvedValue
			return resolvedValue
		})
		if err != nil {
			finalErr = err
			return ""
		}
		return result
	}

	// Resolve each string parameter and store its resolved value in the state
	for name, value := range parameters {
		if s, ok := value.(string); ok {
			state[name] = eval(s)
		} else {
			state[name] = value
		}
	}
	return
}

// EvalString performs variable substitution on the given string, using the
// parameters as allowed values for substitution.
func EvalString(input string, parameters map[string]interface{}) (string, error) {
	if input == "" || !strings.Contains(input, "${") {
		return input, nil
	}
	state := map[string]interface{}{}
	parameters["__input__"] = input
	if err := Eval(state, parameters); err != nil {
		return "", err
	}
	return state["__input__"].(string), nil
}

// EvalStrings performs variable substitution on all the given strings, using
// the parameters as allowed values for substitution.
func EvalStrings(inputs []string, parameters map[string]interface{}) ([]string, error) {
	results := make([]string, 0, len(inputs))
	for _, input := range inputs {
		result, err := EvalString(input, parameters)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// GenericMap transforms a string map to a map of interfaces
func GenericMap(m map[string]string) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range m {
		result[k] = v
	}
	return result
}
