package hash

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSha1(t *testing.T) {
	// echo -n "1234" | shasum
	// 7110eda4d09e062aa5e4a390b0a572ac0d2c0220  -

	var err error
	var value string

	h := SHA1()

	value, err = h.String("1234")
	require.Nil(t, err)
	require.Equal(t, "7110eda4d09e062aa5e4a390b0a572ac0d2c0220", value)

	value, err = h.Object(1234)
	require.Nil(t, err)
	require.Equal(t, "7110eda4d09e062aa5e4a390b0a572ac0d2c0220", value)

	f, err := ioutil.TempFile("", "zim-test-")
	require.Nil(t, err)
	f.Write([]byte("1234"))
	f.Close()

	value, err = h.File(f.Name())
	require.Nil(t, err)
	require.Equal(t, "7110eda4d09e062aa5e4a390b0a572ac0d2c0220", value)
}
