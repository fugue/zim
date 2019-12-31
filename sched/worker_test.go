package sched

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/LuminalHQ/zim/definitions"
	"github.com/LuminalHQ/zim/project"
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
	executor := project.NewBashExecutor()
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

	go worker(ctx, runner, "123", executor, ruleChan, resultChan)

	// Send build rule to the worker
	ruleChan <- build
	close(ruleChan)

	// Receive result from the worker after it runs the rule
	res := <-resultChan
	assert.Equal(t, project.Cached, res.Code)
	assert.Equal(t, "bourgeoisie", res.Error.Error())
	assert.Equal(t, build, res.Rule)
}
