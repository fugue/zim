package sched

import (
	"context"

	"github.com/LuminalHQ/zim/project"
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
	results chan<- *workerResult) {

	for rule := range rules {
		code, err := runner.Run(ctx, rule, project.RunOpts{
			BuildID:  buildID,
			Executor: exec,
		})
		results <- &workerResult{Rule: rule, Code: code, Error: err}
	}
}
