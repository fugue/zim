package hash

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSha256(t *testing.T) {
	// echo -n "1234" | shasum -a 256                                                                                    (docker-desktop/default)
	// 03ac674216f3e15c761ee1a5e255f067953623c8b388b4459e13f978d7c846f4  -

	var err error
	var value string

	expected := "03ac674216f3e15c761ee1a5e255f067953623c8b388b4459e13f978d7c846f4"

	h := SHA256()

	value, err = h.String("1234")
	require.Nil(t, err)
	require.Equal(t, expected, value)

	value, err = h.Object(1234)
	require.Nil(t, err)
	require.Equal(t, expected, value)

	f, err := ioutil.TempFile("", "zim-test-")
	require.Nil(t, err)
	f.Write([]byte("1234"))
	f.Close()

	value, err = h.File(f.Name())
	require.Nil(t, err)
	require.Equal(t, expected, value)
}
