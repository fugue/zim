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
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/fugue/zim/graph"
	"github.com/fugue/zim/project"
	"github.com/hashicorp/go-multierror"
)

// Status indicates the running state of a rule in the scheduler
type Status int

const (

	// Unscheduled says that Rule has not had a chance to run yet
	Unscheduled Status = iota

	// Running indicates the Rule is currently running
	Running

	// Error indicates that Rule execution was attempted but had an error
	Error

	// Completed indicates the Rule ran successfully
	Completed
)

// NewGraphScheduler returns a default scheduler
func NewGraphScheduler() Scheduler {
	return &dagScheduler{}
}

type dagScheduler struct{}

func (s *dagScheduler) Run(ctx context.Context, opts Options) error {

	if opts.NumWorkers < 1 {
		opts.NumWorkers = 1
	}

	// Create a dependency graph which includes the chosen rules and all
	// their transitive dependencies
	schedGraph := project.GraphFromRules(opts.Rules)

	// TODO
	// content, err := dot.Marshal(schedGraph.Graph, "", "", "\t")
	// if err != nil {
	// 	log.Fatal("Could not marshall: ", err)
	// }
	content, err := schedGraph.Marshal()
	if err != nil {
		log.Fatal("Could not marshall: ", err)
	}

	fmt.Println(string(content))
	os.Exit(100)

	executor := opts.Executor
	if executor == nil {
		executor = project.NewBashExecutor()
	}

	// Run the specified number of workers to run rules in parallel
	var wg sync.WaitGroup
	var errors *multierror.Error
	jobs := make(chan *project.Rule)
	results := make(chan *workerResult, opts.NumWorkers)
	for w := 0; w < opts.NumWorkers; w++ {
		wg.Add(1)
		go worker(ctx, opts.Runner, opts.BuildID, executor, jobs, results, &wg)
	}

	// Signal to the workers to exit when done running and then wait
	// for them to exit before returning
	defer func() {
		close(jobs)
		wg.Wait()
	}()

	// Tracks the state of each rule
	ruleStates := map[*project.Rule]Status{}

	// Initialize the ruleStates map with all relevant rules "unscheduled"
	schedGraph.Visit(func(n graph.Node) bool {
		ruleStates[n.(*project.Rule)] = Unscheduled
		return true
	})

	rulesFinished := 0
	rulesCount := len(ruleStates)

	// Called each time a rule starts executing to update scheduler state
	ruleStart := func(r *project.Rule) {
		if ruleStates[r] != Unscheduled {
			panic(fmt.Sprintf("Rule started from unexpected state"))
		}
		ruleStates[r] = Running
	}

	// Called each time a rule finishes executing to update scheduler
	// state. We can update the rule dependency graph as needed here.
	var ruleDone func(*project.Rule, error)
	ruleDone = func(r *project.Rule, err error) {
		rulesFinished++
		if err != nil {
			errors = multierror.Append(errors, err)
			ruleStates[r] = Error
			// Any rules dependent on this rule should now error as well.
			// This recursively calls ruleDone to propagate this error.
			for _, other := range schedGraph.To(r) {
				nextErr := fmt.Errorf("Rule %s failed due to error on dependency %s",
					project.Bright(other.NodeID()), project.Bright(r.NodeID()))
				ruleDone(other.(*project.Rule), nextErr)
			}
		} else {
			ruleStates[r] = Completed
		}
		// By removing the rule from the scheduling graph, any rules that
		// had it as a dependency now have 1 fewer dependency "from" edges.
		// This may make them eligible to run now.
		schedGraph.Remove(r)
	}

	// Execute rules until no more are available
	for rulesFinished < rulesCount {

		// Receive results from workers. This select waits for up to 20 ms so
		// that we're not in a hard loop here while rules are running.
		select {
		case result := <-results:
			ruleDone(result.Rule, result.Error)
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}

		// Pick rules in the graph that have no remaining "from" edges.
		// These are nodes that have no dependencies or the dependencies
		// have already been run and removed from the graph.
		candidateNodes := schedGraph.Filter(func(n graph.Node) bool {
			status := ruleStates[n.(*project.Rule)]
			return status == Unscheduled && len(schedGraph.From(n)) == 0
		})

		if len(candidateNodes) == 0 {
			// No rules ready for scheduling at the moment
			continue
		}

		// These rules are able to execute now
		candidates := nodesToRules(candidateNodes)
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].NodeID() < candidates[j].NodeID()
		})

		// Send rules to workers to execute (non-blocking send)
		var allWorkersBusy bool
		for _, rule := range candidates {
			select {
			case jobs <- rule:
				ruleStart(rule)
			default: // Workers busy
				allWorkersBusy = true
			}
			if allWorkersBusy {
				break
			}
		}
	}

	// Confirm all requested rules executed or produce an error
	for _, rule := range opts.Rules {
		state := ruleStates[rule]
		if state == Unscheduled {
			err := fmt.Errorf("Rule did not run: %s", rule.NodeID())
			errors = multierror.Append(errors, err)
		}
	}

	return errors.ErrorOrNil()
}

func nodesToRules(nodes []graph.Node) (result []*project.Rule) {
	for _, n := range nodes {
		result = append(result, n.(*project.Rule))
	}
	return
}
