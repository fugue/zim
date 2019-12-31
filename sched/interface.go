package sched

import (
	"context"

	"github.com/LuminalHQ/zim/project"
)

// Options used to configure the Scheduler
type Options struct {
	BuildID    string
	Name       string
	Runner     project.Runner
	Executor   project.Executor
	Rules      []*project.Rule
	RunRemote  bool
	NumWorkers int
}

// Scheduler for jobs
type Scheduler interface {

	// Run specified Rules
	Run(context.Context, Options) error
}
