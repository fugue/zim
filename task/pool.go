package task

import (
	"context"
	"time"

	"github.com/hashicorp/go-multierror"
)

func worker(ctx context.Context, id int, runner Runner, jobs <-chan *Options, results chan<- error) {

	for j := range jobs {

		var err error
		var task *Task

		for i := 0; i < 3; i++ {
			task, err = runner.Run(ctx, *j)
			if err != nil {
				time.Sleep(10 * time.Second)
				continue
			}
			break
		}
		if err != nil {
			results <- err
			continue
		}

		for i := 0; i < 3; i++ {
			err = runner.WaitUntilStopped(ctx, task)
			if err != nil {
				time.Sleep(10 * time.Second)
				continue
			}
			break
		}
		results <- err
	}
}

// RunAll runs a task for each configuration and waits for all tasks to finish
func RunAll(ctx context.Context, runner Runner, taskOptions []*Options, batchSize int) error {

	numTasks := len(taskOptions)
	if numTasks == 0 {
		return nil
	}

	jobs := make(chan *Options, numTasks)
	results := make(chan error, numTasks)

	numWorkers := batchSize
	if numWorkers > numTasks {
		numWorkers = numTasks
	}

	for w := 0; w < numWorkers; w++ {
		go worker(ctx, w, runner, jobs, results)
	}
	for _, opt := range taskOptions {
		jobs <- opt
	}
	close(jobs)

	var result *multierror.Error
	for i := 0; i < numTasks; i++ {
		taskResult := <-results
		result = multierror.Append(result, taskResult)
	}
	return result.ErrorOrNil()
}
