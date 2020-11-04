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
package project

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/fugue/zim/definitions"
)

var ignoreDirs map[string]bool

func init() {
	ignoreDirs = map[string]bool{
		".git":         true,
		"node_modules": true,
		"build":        true,
		"artifacts":    true,
		"dist":         true,
		"venv":         true,
		"venvs":        true,
		".mypy_cache":  true,
		".cache":       true,
		".npm":         true,
		".stack-work":  true,
		"vendor":       true,
	}
}

func fileExists(p string) bool {
	if _, err := os.Stat(p); err == nil {
		return true
	}
	return false
}

func discoverDefs(root string) ([]string, error) {

	var paths []string

	callback := func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fileInfo.IsDir() {
			if ignore := ignoreDirs[filepath.Base(filePath)]; ignore {
				return filepath.SkipDir
			}
		}
		if fileInfo.Name() != "component.yaml" && fileInfo.Name() != "zim.yaml" {
			return nil
		}
		paths = append(paths, filePath)
		return nil
	}

	if err := filepath.Walk(root, callback); err != nil {
		return paths, fmt.Errorf("Failed to walk %s: %s", root, err)
	}
	return paths, nil
}

// Discover Components located within the given directory. The directory
// structure is searched recursively. Returns loaded Component definitions.
func Discover(root string) (*definitions.Project, []*definitions.Component, error) {

	var err error
	var componentPatterns []string
	var pDef *definitions.Project

	projectDefPath := path.Join(root, ".zim", "project.yaml")
	if fileExists(projectDefPath) {
		pDef, err = definitions.LoadProjectFromPath(projectDefPath)
		if err != nil {
			return nil, nil, fmt.Errorf("Invalid project.yaml: %s", err)
		}
		if len(pDef.Components) > 0 {
			componentPatterns = pDef.Components
		}
	}

	var paths []string

	if len(componentPatterns) == 0 {
		paths, err = discoverDefs(root)
		if err != nil {
			return nil, nil, err
		}
	} else {
		for _, pattern := range componentPatterns {
			matches, err := Glob(filepath.Join(root, pattern))
			if err != nil {
				return nil, nil, err
			}
			for _, match := range matches {
				paths = append(paths, match)
			}
		}
	}

	templateDir := path.Join(root, ".zim", "templates")
	nameUsed := map[string]bool{}
	templates := map[string]*definitions.Component{}
	var defs []*definitions.Component

	// Load base templates first if they exist
	tmplInfos, _ := ioutil.ReadDir(templateDir)
	for _, info := range tmplInfos {
		if !strings.HasSuffix(info.Name(), "yaml") {
			continue
		}
		defPath := path.Join(templateDir, info.Name())
		def, err := definitions.LoadComponentFromPath(defPath)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to load template %s: %s", defPath, err)
		}
		if def.Kind == "" {
			return nil, nil, fmt.Errorf("Template kind unset: %s", defPath)
		}
		templates[def.Kind] = def
	}

	for _, defPath := range paths {
		def, err := definitions.LoadComponentFromPath(defPath)
		if err != nil {
			return nil, nil, fmt.Errorf("Invalid component %s: %s", defPath, err)
		}
		// Ignore components by request
		if def.Ignore {
			continue
		}
		// Require component name to be filled in
		if def.Name == "" {
			return nil, nil, fmt.Errorf("Component name unset in %s", defPath)
		}
		// Disallow duplicate component names
		if _, used := nameUsed[def.Name]; used {
			return nil, nil, fmt.Errorf("Duplicate component name: %s", def.Name)
		}
		nameUsed[def.Name] = true
		// Raise error if definition kind is unknown
		tmpl, found := templates[def.Kind]
		if def.Kind != "" && !found {
			return nil, nil, fmt.Errorf("Component kind unknown %s: %s", defPath, def.Kind)
		}
		if tmpl != nil {
			defs = append(defs, tmpl.Merge(def))
		} else {
			defs = append(defs, def)
		}
	}
	return pDef, defs, nil
}
