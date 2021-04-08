package envsub

import (
	"fmt"

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
