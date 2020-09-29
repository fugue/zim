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

import "time"

// Message sent to give updates on progress
type Message struct {
	BuildID       string    `json:"build_id"`
	Bucket        string    `json:"bucket"`
	OutputKey     string    `json:"output_key"`
	ResultKey     string    `json:"result_key"`
	SourceKey     string    `json:"source_key"`
	CommitID      string    `json:"commit_id"`
	Component     string    `json:"component"`
	Rule        string    `json:"rule"`
	Time          time.Time `json:"time"`
	Exit          bool      `json:"exit"`
	ResponseQueue string    `json:"response_queue"`
	Output        string    `json:"output"`
	Error         string    `json:"error"`
	Worker        string    `json:"worker"`
	Kind          string    `json:"kind"`
	Text          string    `json:"text"`
	Success       bool      `json:"success"`
}
