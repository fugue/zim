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
package store

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
)

// NotFound indicates an object does not exist
type NotFound string

func (e NotFound) Error() string { return string(e) }

// Item in storage
type Item struct {
	Key          string
	ETag         string
	Version      string
	Size         int64
	LastModified time.Time
	Meta         map[string]string
}

// Exists returns true if it is a valid Item
func (item Item) Exists() bool {
	return item.Key != "" && item.ETag != "" && item.Size > 0
}

// Store is an interface to Get and Put items into storage
type Store interface {

	// Get an item from the Store
	Get(ctx context.Context, key, dst string) error

	// Put an item in the Store
	Put(ctx context.Context, key, src string, meta map[string]string) error

	// List contents of the Store
	List(ctx context.Context, prefix string) ([]string, error)

	// Head checks if the item exists in the store
	Head(ctx context.Context, key string) (Item, error)
}

// Returns the size and last modified timestamp for the file at the given path
func fileStat(name string) (int64, time.Time) {
	info, err := os.Stat(name)
	if err != nil {
		return 0, time.Time{}
	}
	return info.Size(), info.ModTime()
}

func isNotFound(err error) bool {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case "NotFound":
			return true
		}
	}
	return false
}
