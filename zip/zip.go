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
package zip

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Unzip a zip file to the given destination directory
func Unzip(src, dstDir string) error {

	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("Failed to open zip file: %s", err)
	}
	defer func() { r.Close() }()

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("Failed to create dst dir: %s", err)
	}

	// Closure to address file descriptors issue with all the deferred Close()
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() { rc.Close() }()

		path := filepath.Join(dstDir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() { f.Close() }()

			if _, err = io.Copy(f, rc); err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		if err := extractAndWriteFile(f); err != nil {
			return fmt.Errorf("Failed to extract %s: %s", f.Name, err)
		}
	}
	return nil
}

// Zip a directory
func Zip(src, dst string) error {

	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("Failed to create zip file %s: %s", dst, err)
	}
	defer f.Close()

	archive := zip.NewWriter(f)
	defer archive.Close()

	// info, err := os.Stat(src)
	// if err != nil {
	// 	return fmt.Errorf("Failed to stat input %s: %s", src, err)
	// }

	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		// if baseDir != "" {
		// 	header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, src))
		// }
		header.Name = strings.TrimPrefix(path, src)
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}
