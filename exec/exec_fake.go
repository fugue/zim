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

package exec

import (
	"context"
)

// FakeExecutor wraps another Executor and allows overriding the UsesDocker result
type FakeExecutor struct {
	Docker  bool
	Wrapped Executor
}

func (e *FakeExecutor) Execute(ctx context.Context, opts ExecOpts) error {
	return e.Wrapped.Execute(ctx, opts)
}

func (e *FakeExecutor) UsesDocker() bool {
	return e.Docker
}

func (e *FakeExecutor) ExecutorPath(hostPath string) (string, error) {
	return e.Wrapped.ExecutorPath(hostPath)
}
