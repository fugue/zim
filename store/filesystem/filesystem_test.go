package filesystem

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/fugue/zim/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Confirm Head and Get requests on a unknown key result in a
// "not found" error
func TestMissingKey(t *testing.T) {

	var err error
	var meta store.ItemMeta
	ctx := context.Background()

	cacheDir, err := ioutil.TempDir("", "zim-test-")
	require.Nil(t, err)
	defer os.RemoveAll(cacheDir)

	f, err := ioutil.TempFile("", "zim-tmp-file-")
	require.Nil(t, err)
	f.Close()
	defer os.RemoveAll(f.Name())

	fs := New(cacheDir)

	meta, err = fs.Head(ctx, "/missing/key")
	assert.Equal(t, "not found: /missing/key", err.Error())
	assert.Equal(t, 0, len(meta.Meta))

	err = fs.Get(ctx, "/missing/key", f.Name())
	assert.Equal(t, "not found: /missing/key", err.Error())
	assert.Equal(t, 0, len(meta.Meta))
}

// Confirm that a Get works on a file we put in the store manually
func TestPresentKey(t *testing.T) {

	inputFile := "test_fixture1.txt"
	outputFile := "test_fixture1_output.txt"
	key := "abcdef"

	ctx := context.Background()

	cacheDir, err := ioutil.TempDir("", "zim-test-")
	require.Nil(t, err)
	defer os.RemoveAll(cacheDir)

	// Remove output file in case a previous test run created it
	os.RemoveAll(outputFile)

	fs := New(cacheDir)

	// Add a test fixture file to the store
	require.Nil(t, fs.Put(ctx, key, inputFile, map[string]string{"foo": "bar"}))

	// Key abcdef should be nested at <cache>/ab/cd/abcdef
	_, err = os.Stat(filepath.Join(cacheDir, "ab", "cd", key))
	require.Nil(t, err)

	// Confirm Head succeeds and returns the item metadata
	item, err := fs.Head(ctx, key)
	require.Nil(t, err)
	require.Equal(t, map[string]string{"foo": "bar"}, item.Meta)

	// Retrieve the item and store it in the local directory as test_get.txt
	require.Nil(t, fs.Get(ctx, key, outputFile))

	// Confirm the resulting file exists and has the expected contents
	bytes, err := ioutil.ReadFile(outputFile)
	require.Nil(t, err)
	require.Equal(t, "The quick brown fox\njumps over the lazy dog", string(bytes))
}
