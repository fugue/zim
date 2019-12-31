package project

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscover(t *testing.T) {

	dir := testDir()

	testComponentDir(dir, "hammer")
	testComponentDir(dir, "nail")

	pDef, defs, err := Discover(dir)
	require.Nil(t, pDef)
	require.Nil(t, err)
	require.Len(t, defs, 2)

	def0 := defs[0]
	def1 := defs[1]

	assert.Equal(t, "hammer", def0.Name)
	assert.Equal(t, "nail", def1.Name)

	assert.Equal(t, path.Join(dir, "src", "hammer", "component.yaml"), def0.Path)
	assert.Equal(t, path.Join(dir, "src", "nail", "component.yaml"), def1.Path)
}
