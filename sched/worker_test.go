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
	"errors"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/fugue/zim/definitions"
	"github.com/fugue/zim/exec"
	"github.com/fugue/zim/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDir() string {
	dir, err := ioutil.TempDir("", "zim-")
	if err != nil {
		panic(err)
	}
	return dir
}

func TestWorker(t *testing.T) {

	ctx := context.Background()
	executor := exec.NewBashExecutor()
	ruleChan := make(chan *project.Rule, 1)
	resultChan := make(chan *workerResult, 1)

	dir := testDir()
	defer os.RemoveAll(dir)

	def := &definitions.Component{
		Path: path.Join(dir, "widget"),
		Name: "widget",
		Rules: map[string]definitions.Rule{
			"build": definitions.Rule{},
		},
	}

	p, err := project.NewWithOptions(project.Opts{
		Root:          dir,
		ComponentDefs: []*definitions.Component{def},
	})
	require.Nil(t, err)

	c := p.Components().WithName("widget").First()
	require.NotNil(t, c)

	build, found := c.Rule("build")
	require.True(t, found)

	runner := project.RunnerFunc(func(ctx context.Context, rule *project.Rule, opts project.RunOpts) (project.Code, error) {
		assert.Equal(t, build, rule)
		assert.Equal(t, "123", opts.BuildID)
		assert.Equal(t, executor, opts.Executor)
		assert.Equal(t, nil, opts.Output)
		return project.Cached, errors.New("bourgeoisie")
	})

	var wg sync.WaitGroup
	wg.Add(1)

	go worker(ctx, runner, "123", executor, ruleChan, resultChan, &wg)

	// Send build rule to the worker
	ruleChan <- build
	close(ruleChan)

	// Receive result from the worker after it runs the rule
	res := <-resultChan
	assert.Equal(t, project.Cached, res.Code)
	assert.Equal(t, "bourgeoisie", res.Error.Error())
	assert.Equal(t, build, res.Rule)

	wg.Wait()
}
