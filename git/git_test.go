// Copyright 2020 Fugue, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package git

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/fugue/zim/zip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommitID(t *testing.T) {
	ctx := context.Background()
	commit, err := CommitID(ctx, ".")
	require.Nil(t, err, "Failed to read commit ID")
	assert.Len(t, commit, 10, "Expected commit ID string of length 10")
}

func TestRepoRoot(t *testing.T) {
	root, err := RepoRoot(".")
	require.Nil(t, err, "Failed to read repo root")
	assert.True(t, filepath.IsAbs(root), "Expected absolute path")
}

func TestCreateArchive(t *testing.T) {
	ctx := context.Background()

	root, err := RepoRoot(".")
	require.Nil(t, err, "Failed to read repo root")

	tmpDir, err := ioutil.TempDir("", "zim-test-")
	require.Nil(t, err)
	defer os.RemoveAll(tmpDir)

	repoZip := path.Join(tmpDir, "myrepo.zip")
	require.Nil(t, CreateArchive(ctx, root, repoZip))

	extractDir, err := ioutil.TempDir("", "zim-test-")
	require.Nil(t, err)
	defer os.RemoveAll(extractDir)

	require.Nil(t, zip.Unzip(repoZip, extractDir))

	files, err := ioutil.ReadDir(extractDir)
	require.Nil(t, err)

	foundFiles := map[string]bool{}
	foundDirs := map[string]bool{}
	for _, f := range files {
		if f.IsDir() {
			foundDirs[f.Name()] = true
		} else {
			foundFiles[f.Name()] = true
		}
	}

	assert.True(t, foundFiles["go.mod"], "Expected go.mod to exist")
	assert.True(t, foundDirs["git"], "Expected git directory to exist")
}
