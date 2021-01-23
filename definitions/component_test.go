package definitions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeComponents(t *testing.T) {

	a := &Component{
		Name: "",
		Kind: "go",
		Toolchain: Toolchain{
			Items: []ToolchainItem{
				ToolchainItem{
					Name:    "go version output",
					Command: "go version",
				},
			},
		},
		Exports: map[string]Export{
			"source": Export{
				Provider:  "FOO",
				Resources: []string{"*.go", "*.json"},
				Ignore:    []string{"*_test.go"},
			},
		},
	}

	b := &Component{
		Kind: "go",
		Name: "flubber",
		Exports: map[string]Export{
			"source": Export{
				Resources: []string{"*.go"},
			},
		},
	}

	merged := a.Merge(b)

	assert.Equal(t, "flubber", merged.Name)
	assert.Equal(t, "go", merged.Kind)
	require.Len(t, merged.Exports, 1)

	source, ok := merged.Exports["source"]
	require.True(t, ok)

	assert.Equal(t, "FOO", source.Provider)
	assert.Equal(t, []string{"*_test.go"}, source.Ignore)
	assert.Equal(t, []string{"*.go"}, source.Resources)
}
