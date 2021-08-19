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
package sched

import (
	"context"
	"sync"

	"github.com/fugue/zim/exec"
	"github.com/fugue/zim/project"
)

// workerResult contains the results of running a rule
type workerResult struct {
	Rule  *project.Rule
	Code  project.Code
	Error error
}

// worker may be run in a goroutine to grab jobs off the "rules" channel
// and execute them one after the other
func worker(
	ctx context.Context,
	runner project.Runner,
	buildID string,
	exc exec.Executor,
	rules <-chan *project.Rule,
	results chan<- *workerResult,
	wg *sync.WaitGroup) {

	defer wg.Done()

	for {
		select {

		case rule, ok := <-rules:
			if !ok {
				return
			}
			code, err := runner.Run(ctx, rule, project.RunOpts{
				BuildID:  buildID,
				Executor: exc,
			})
			if ctx.Err() == nil {
				results <- &workerResult{Rule: rule, Code: code, Error: err}
			}

		case <-ctx.Done():
			return
		}
	}
}
