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
)

// NotFound indicates an object does not exist
type NotFound string

func (e NotFound) Error() string { return string(e) }

// ItemMeta contains metadata for an item in storage
type ItemMeta struct {
	Meta map[string]string `json:"meta"`
}

// Store is an interface to Get and Put items into storage
type Store interface {

	// Get an item from the Store
	Get(ctx context.Context, key, dst string) error

	// Put an item in the Store
	Put(ctx context.Context, key, src string, meta map[string]string) error

	// Head checks if the item exists in the store
	Head(ctx context.Context, key string) (ItemMeta, error)
}
