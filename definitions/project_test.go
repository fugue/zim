package definitions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironment(t *testing.T) {
	obj := []interface{}{
		"foo",
		"bar=fizzle biz",
		map[interface{}]interface{}{
			"ACCOUNT": map[interface{}]interface{}{
				"run": "aws sts get-caller-identity",
			},
		},
	}
	env, err := GetEnvironment(obj)
	require.Nil(t, err)
	require.Len(t, env.Variables, 3)

	var1 := env.Variables[0]
	var2 := env.Variables[1]
	var3 := env.Variables[2]

	assert.Equal(t, "foo", var1.Definition)
	assert.Equal(t, "bar=fizzle biz", var2.Definition)
	assert.Equal(t, "ACCOUNT", var3.Definition)
	assert.Equal(t, "aws sts get-caller-identity", var3.Script)
}
