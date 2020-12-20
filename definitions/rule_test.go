package definitions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeRule(t *testing.T) {

	a := Rule{
		Inputs:  []string{"main.go"},
		Outputs: []string{"out"},
		Requires: []Dependency{
			Dependency{
				Rule: "some-other-rule",
			},
		},
		Native: true,
		Commands: []interface{}{
			map[string]interface{}{"run": "echo HELLO"},
		},
	}

	b := Rule{
		Command: "echo GOODBYE",
	}

	merged := mergeRule(a, b)

	assert.Equal(t, []string{"main.go"}, merged.Inputs)
	assert.Equal(t, []string{"out"}, merged.Outputs)
	assert.Equal(t, []Dependency{
		Dependency{
			Rule: "some-other-rule",
		},
	}, merged.Requires)
	assert.Equal(t, true, merged.Native)
	assert.Nil(t, merged.Commands)
	assert.Equal(t, "echo GOODBYE", merged.Command)
}
