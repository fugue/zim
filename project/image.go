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
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// DockerError is a simple error type
type DockerError string

func (e DockerError) Error() string { return string(e) }

// ImageNotFound indicates an image doesn't exist that matches a search
const ImageNotFound = DockerError("Image not found")

// Docker is a type used to interact with the Docker daemon. It is used to
// work with Docker images as Zim Resources. This implements the Provider
// Go interface.
type Docker struct {
	cli *client.Client
}

// NewDocker returns a Provider for Docker images.
func NewDocker() (Provider, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv,
		client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Docker{cli: cli}, nil
}

// Init accepts configuration options from Project configuration
func (d *Docker) Init(opts map[string]interface{}) error {
	return nil
}

// Name identifies the type of the Docker Provider
func (d *Docker) Name() string {
	return "docker"
}

// New returns a Docker image Resource where "path" is interpreted as an
// image name or ID. The image may or may not exist currently.
func (d *Docker) New(path string) Resource {
	return &Image{docker: d, name: path}
}

// Match existing Docker images according to the given pattern.
// Example patterns: "foo", "foo:latest", etc.
func (d *Docker) Match(pattern string) (r Resources, err error) {
	ctx := context.Background()
	images, err := d.FindImages(ctx, pattern)
	if err != nil {
		return nil, err
	}
	r = make(Resources, 0, len(images))
	for _, summary := range images {
		r = append(r, &Image{
			docker:  d,
			name:    pattern,
			summary: summary,
		})
	}
	return
}

// FindImages finds an image with the given name. This may or may not include
// an image tag.
func (d *Docker) FindImages(ctx context.Context, name string) ([]types.ImageSummary, error) {
	summaries, err := d.cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return nil, err
	}
	var images []types.ImageSummary
	if !strings.Contains(name, ":") {
		name = fmt.Sprintf("%s:latest", name)
	}
	for _, summary := range summaries {
		// RepoTags looks like ["foo:latest", "foo:tag"]
		if matchDockerTags(name, summary.RepoTags) {
			images = append(images, summary)
		}
	}
	if len(images) > 0 {
		return images, nil
	}
	return nil, ImageNotFound
}

func matchDockerTags(pattern string, tags []string) bool {
	for _, tag := range tags {
		tagParts := strings.Split(tag, ":")
		if len(tagParts) != 2 {
			panic(fmt.Sprintf("Unexpected docker image name format: %s", tag))
		}
		if pattern == tagParts[0] || pattern == tag {
			return true
		}
	}
	return false
}

// Image implements the Resource interface for Docker images
type Image struct {
	docker  *Docker
	name    string
	summary types.ImageSummary
}

// OnFilesystem is false for Doc
func (img *Image) OnFilesystem() bool {
	return false
}

// Cacheable is false for Images - not implemented yet
func (img *Image) Cacheable() bool {
	return false
}

// Name of the Resource
func (img *Image) Name() string {
	return img.name
}

// Path returns the absolute path to the Image
func (img *Image) Path() string {
	return img.name
}

// Exists indicates whether the Image currently exists
func (img *Image) Exists() (bool, error) {
	ctx := context.Background()
	summaries, err := img.docker.FindImages(ctx, img.name)
	if err != nil {
		return false, err
	}
	if len(summaries) == 0 {
		return false, nil
	}
	return true, nil
}

// Hash for Docker images returns the image ID
func (img *Image) Hash() (string, error) {
	ctx := context.Background()
	summaries, err := img.docker.FindImages(ctx, img.name)
	if err != nil {
		return "", err
	}
	if len(summaries) == 0 {
		return "", fmt.Errorf("Image not found: %s", img.name)
	}
	return summaries[0].ID, nil
}

// LastModified time of this Docker image. This corresponds to the image
// build time. It is not updated when a docker build detects that an image
// already exists.
func (img *Image) LastModified() (time.Time, error) {
	ctx := context.Background()
	summaries, err := img.docker.FindImages(ctx, img.name)
	if err != nil {
		return time.Time{}, err
	}
	if len(summaries) == 0 {
		return time.Time{}, fmt.Errorf("Image not found: %s", img.name)
	}
	return time.Unix(summaries[0].Created, 0), nil
}

// AsFile returns the path to the file
func (img *Image) AsFile() (string, error) {
	return "", errors.New("unsupported")
}
