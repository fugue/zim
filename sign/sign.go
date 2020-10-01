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
package sign

import "time"

// Input for a signing request
type Input struct {
	Method        string            `json:"method"`
	Name          string            `json:"name"`
	Metadata      map[string]string `json:"metadata"`
	ContentLength int64             `json:"content_len"`
}

// Output from a signing request
type Output struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// Item contains information about an item in storage
type Item struct {
	Key          string            `json:"key"`
	Metadata     map[string]string `json:"metadata"`
	Version      string            `json:"version"`
	ETag         string            `json:"etag"`
	Size         int64             `json:"size"`
	LastModified time.Time         `json:"last_modified"`
}
