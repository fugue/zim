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
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/fugue/zim/project"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}

func getRepository(dir string) (string, error) {
	if dir == "" {
		dir = "."
	}
	return repoRoot(dir)
}

func gitRoot(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return getRepository(absDir)
}

func getProject(dir string) (*project.Project, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if repo, err := getRepository(absDir); err == nil {
		absDir = repo
	}
	return project.New(absDir)
}

type zimOptions struct {
	Directory  string
	URL        string
	Region     string
	Cache      string
	UseDocker  bool
	Kinds      []string
	Components []string
	Rules      []string
	Debug      bool
	OutputMode string
	Jobs       int
	CacheMode  string
	Token      string
}

func getZimOptions(cmd *cobra.Command, args []string) zimOptions {
	opts := zimOptions{
		Directory:  viper.GetString("dir"),
		URL:        viper.GetString("url"),
		Region:     viper.GetString("region"),
		Cache:      viper.GetString("cache"),
		Kinds:      viper.GetStringSlice("kinds"),
		Components: viper.GetStringSlice("components"),
		Rules:      viper.GetStringSlice("rules"),
		UseDocker:  viper.GetBool("docker"),
		Debug:      viper.GetBool("debug"),
		OutputMode: viper.GetString("output"),
		Jobs:       viper.GetInt("jobs"),
		CacheMode:  viper.GetString("cache"),
		Token:      viper.GetString("token"),
	}
	if opts.Cache == "" {
		opts.Cache = XDGCache()
	}
	// Strip paths to components if provided, e.g. src/foo -> foo
	for i, c := range opts.Components {
		opts.Components[i] = filepath.Base(c)
	}

	// Rules can be specified by arguments or options for run
	if cmd.Name() == "run" && len(opts.Rules) == 0 && len(args) > 0 {
		opts.Rules = args
	}

	return opts
}

// XDGCache returns the local cache directory
func XDGCache() string {
	value := os.Getenv("XDG_CACHE_HOME")
	if value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return path.Join(home, ".cache")
}

func fileExists(p string) bool {
	if _, err := os.Stat(p); err == nil {
		return true
	}
	return false
}

// repoRoot returns the root directory of the Git repository, given any
// path within the repository
func repoRoot(dir string) (string, error) {

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
