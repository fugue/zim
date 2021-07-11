package envsub

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalBasic(t *testing.T) {

	state := map[string]interface{}{}

	params := map[string]interface{}{
		"age":         32,
		"name":        "fred",
		"description": "${name} age ${age}",
	}

	err := Eval(state, params)
	require.Nil(t, err)

	require.Equal(t, state, map[string]interface{}{
		"age":         32,
		"name":        "fred",
		"description": "fred age 32",
	})
}

func TestEvalChain(t *testing.T) {

	state := map[string]interface{}{}

	params := map[string]interface{}{
		"a":      "A",
		"b":      "B",
		"c":      "C",
		"alpha":  "${a} ${b} ${c}",
		"repeat": "|${alpha}|${alpha}|",
	}

	err := Eval(state, params)
	require.Nil(t, err)

	require.Equal(t, state, map[string]interface{}{
		"a":      "A",
		"b":      "B",
		"c":      "C",
		"alpha":  "A B C",
		"repeat": "|A B C|A B C|",
	})
}

func TestEvalTransforms(t *testing.T) {

	state := map[string]interface{}{
		"a": "apple",
		"b": "BANANA",
		"c": "CARROT",
	}

	params := map[string]interface{}{
		"apple":  "Eat an ${a^}",
		"banana": "${b,,} for scale",
		"carrot": "No carrots-${c/CARROT/Broccoli}-instead!",
	}

	err := Eval(state, params)
	require.Nil(t, err)

	require.Equal(t, state, map[string]interface{}{
		"a":      "apple",
		"b":      "BANANA",
		"c":      "CARROT",
		"apple":  "Eat an Apple",
		"banana": "banana for scale",
		"carrot": "No carrots-Broccoli-instead!",
	})
}

func TestEvalRecursion(t *testing.T) {

	state := map[string]interface{}{}

	params := map[string]interface{}{
		"a": "${c}",
		"b": "bravo",
		"c": "${b} ${a}",
	}

	err := Eval(state, params)
	require.NotNil(t, err)
	require.Equal(t, "recursion detected", err.Error())
}

func TestEvalUnknownVariable(t *testing.T) {

	state := map[string]interface{}{}

	params := map[string]interface{}{
		"a": "alpha",
		"b": "bravo",
		"c": "${a} ${b} ${WHAT}",
	}

	err := Eval(state, params)
	require.NotNil(t, err)
	require.Equal(t, "unknown variable: WHAT", err.Error())
}

func TestEvalString(t *testing.T) {
	input := "Prefix ${NAME} ${FOO} Suffix"
	result, err := EvalString(input, map[string]interface{}{
		"FOO":  "FOO",
		"NAME": "EAGLE",
	})
	require.Nil(t, err)
	require.Equal(t, "Prefix EAGLE FOO Suffix", result)
}
