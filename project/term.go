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
	"github.com/fatih/color"
)

var (
	// Bright highlights text in the terminal
	Bright func(args ...interface{}) string

	// Cyan text color
	Cyan func(args ...interface{}) string

	// Green text color
	Green func(args ...interface{}) string

	// Red text color
	Red func(args ...interface{}) string

	// Yellow text color
	Yellow func(args ...interface{}) string
)

func init() {
	Bright = color.New(color.FgHiWhite).SprintFunc()
	Cyan = color.New(color.FgCyan).SprintFunc()
	Green = color.New(color.FgGreen).SprintFunc()
	Red = color.New(color.FgRed).SprintFunc()
	Yellow = color.New(color.FgYellow).SprintFunc()
}
