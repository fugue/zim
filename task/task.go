package task

import (
	"context"
)

// Options configures a task to run
type Options struct {
	Definition  string
	Environment map[string]string
	Memory      int64
	CPU         int64
}

// Task that was run
type Task struct {
	ARN       string `json:"arn"`
	ID        string `json:"id"`
	LogGroup  string `json:"log_group"`
	LogStream string `json:"log_stream"`
}

// Runner is an interface used to run tasks
type Runner interface {
	Run(context.Context, Options) (*Task, error)

	WaitUntilRunning(context.Context, ...*Task) error

	WaitUntilStopped(context.Context, ...*Task) error
}
