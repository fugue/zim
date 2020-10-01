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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/fugue/zim/store"
	"github.com/fugue/zim/zip"
)

// CreateArchive creates a Git archive at the given path
func CreateArchive(ctx context.Context, repo, dst string) error {
	if !strings.HasSuffix(dst, ".zip") {
		return fmt.Errorf("Destination file must end with .zip")
	}
	// Determine whether the git repository has submodules
	submodulePaths, err := ListSubmodules(ctx, repo)
	if err != nil {
		return err
	}
	// This temporary directory will be used to manage zip files
	tmpDir, err := ioutil.TempDir("", "zim-archive-")
	if err != nil {
		return fmt.Errorf("Failed to create tmp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)
	// Create zip of the main git repository
	archive, err := Archive(ctx, repo, tmpDir)
	if err != nil {
		return err
	}
	// We're done if there are no submodules
	if len(submodulePaths) == 0 {
		return os.Rename(archive, dst)
	}
	// Create a zip file for each git submodule
	var submodules []string
	for _, submodule := range submodulePaths {
		combinedPath := path.Join(repo, submodule)
		submoduleArchive, err := Archive(ctx, combinedPath, tmpDir)
		if err != nil {
			return fmt.Errorf("Failed to handle submodule %s: %s", submodule, err)
		}
		submodules = append(submodules, submoduleArchive)
	}
	// Create a new directory to hold files from main repo and all submodules
	combinedDir, err := ioutil.TempDir(tmpDir, "combined-")
	if err != nil {
		return fmt.Errorf("Failed to create tmp dir: %s", err)
	}
	// Unzip all archives into this one directory
	if err := zip.Unzip(archive, combinedDir); err != nil {
		return err
	}
	for i, sPath := range submodulePaths {
		combinedPath := path.Join(combinedDir, sPath)
		if err := os.MkdirAll(combinedPath, 0755); err != nil {
			return err
		}
		if err := zip.Unzip(submodules[i], combinedPath); err != nil {
			return err
		}
	}
	// Create final zip of all files
	if err := zip.Zip(combinedDir, dst); err != nil {
		return fmt.Errorf("Failed to zip combined archive %s: %s", dst, err)
	}
	return nil
}

// InitSubmodules call "git submodule init"
func InitSubmodules(ctx context.Context, gitDir string) error {

	command := exec.CommandContext(ctx, "git", "submodule", "init")
	command.Dir = gitDir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		return fmt.Errorf("Failed to submodule init %s: %s", gitDir, err)
	}
	return nil
}

// ListSubmodules returns a list of paths to submodules within a git repo
func ListSubmodules(ctx context.Context, gitDir string) ([]string, error) {

	var buf bytes.Buffer
	command := exec.CommandContext(ctx, "git", "submodule", "status")
	command.Dir = gitDir
	command.Stdout = &buf
	command.Stderr = &buf

	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("Failed to get submodule status of %s: %s",
			gitDir, err)
	}
	var submodules []string
	for _, line := range strings.Split(buf.String(), "\n") {
		parts := strings.Split(strings.TrimSpace(line), " ")
		if len(parts) < 2 {
			continue
		}
		submodule := strings.TrimSpace(parts[1])
		submodules = append(submodules, submodule)
	}
	return submodules, nil
}

// Archive a single git repository at the given path
func Archive(ctx context.Context, gitDir, tmpDir string) (string, error) {

	tmpFile, err := ioutil.TempFile(tmpDir, "zim.*.zip")
	if err != nil {
		return "", fmt.Errorf("Failed to create archive file: %s", err)
	}
	tmpName := tmpFile.Name()
	tmpFile.Close()

	args := []string{"archive", "-o", tmpName, "HEAD"}
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = gitDir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		return "", fmt.Errorf("Failed to create git archive %s: %s", tmpName, err)
	}
	return tmpName, nil
}

// CommitID returns a shortened commit ID for the given Git repository
func CommitID(ctx context.Context, gitDir string) (string, error) {

	var b bytes.Buffer

	args := []string{"rev-parse", "--short=10", "HEAD"}
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = gitDir
	command.Stdout = &b
	command.Stderr = &b

	if err := command.Run(); err != nil {
		errMsg := b.String()
		return "", fmt.Errorf("Failed to get commit ID in dir %s: %s - %s",
			gitDir, err, errMsg)
	}
	return strings.TrimSpace(b.String()), nil
}

// DownloadExtractArchive downloads a Zip of a Git repo and unzips it
func DownloadExtractArchive(ctx context.Context, store store.Store, workspace, key string) error {

	tmpFile, err := ioutil.TempFile("", "zim.*.zip")
	if err != nil {
		return fmt.Errorf("Failed to create tmp file: %s", err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)
	tmpFile.Close()

	if err := store.Get(ctx, key, tmpName); err != nil {
		return fmt.Errorf("Failed to download archive %s: %s", key, err)
	}
	if err := zip.Unzip(tmpName, workspace); err != nil {
		return fmt.Errorf("Failed to extract repo %s: %s", tmpName, err)
	}
	return nil
}

// RepoRoot returns the root directory of the Git repository, given any
// path within the repository
func RepoRoot(dir string) (string, error) {

	var b bytes.Buffer
	args := []string{"rev-parse", "--git-dir"}
	command := exec.Command("git", args...)
	command.Dir = dir
	command.Stdout = &b
	command.Stderr = &b

	if err := command.Run(); err != nil {
		return "", fmt.Errorf("Failed to run git rev-parse: %s", err)
	}
	output := strings.TrimSpace(b.String())
	if output == ".git" {
		return dir, nil
	}
	return filepath.Dir(output), nil
}
