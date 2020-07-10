package sched

import (
	"context"
	"sync"

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
	exec project.Executor,
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
				Executor: exec,
			})
			if ctx.Err() == nil {
				results <- &workerResult{Rule: rule, Code: code, Error: err}
			}

		case <-ctx.Done():
			return
		}
	}
}
