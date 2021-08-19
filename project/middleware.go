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
package project

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// BufferedOutput is middleware that shows rule stdout and stderr
func BufferedOutput(runner Runner) Runner {
	return RunnerFunc(func(ctx context.Context, r *Rule, opts RunOpts) (Code, error) {
		buffer := &bytes.Buffer{}
		opts.Output = buffer
		opts.DebugOutput = buffer

		code, err := runner.Run(ctx, r, opts)

		output := strings.TrimSpace(buffer.String())
		if len(output) > 0 {
			for _, line := range strings.Split(output, "\n") {
				fmt.Println(line)
			}
		}
		return code, err
	})
}

// Logger is middleware that wraps logging around Rule execution
func Logger(runner Runner) Runner {
	return RunnerFunc(func(ctx context.Context, r *Rule, opts RunOpts) (Code, error) {

		if opts.Output == nil {
			opts.Output = os.Stdout
		}

		fmt.Fprintln(opts.Output, "rule:", Bright(r.NodeID()))
		startedAt := time.Now()

		code, err := runner.Run(ctx, r, opts)

		if code == Skipped {
			fmt.Fprintln(opts.Output, "rule:", Bright(r.NodeID()), Green("[SKIPPED]"))
			return code, err
		} else if code == Cached {
			fmt.Fprintln(opts.Output, "rule:", Bright(r.NodeID()), Green("[CACHED]"))
			return code, err
		}

		duration := time.Since(startedAt)
		durationStr := fmt.Sprintf("in %.3f sec", duration.Seconds())

		if err != nil {
			isKilled := strings.Contains(err.Error(), "signal: killed")
			isCanceled := strings.Contains(err.Error(), "context canceled")

			if isKilled || isCanceled {
				fmt.Fprintln(opts.Output, "rule:", Bright(r.NodeID()),
					Bright(durationStr), Red("[KILLED]"))
			} else {
				fmt.Fprintln(opts.Output, "rule:", Bright(r.NodeID()),
					Bright(durationStr), Red("[FAILED]"))
			}
		} else {
			fmt.Fprintln(opts.Output, "rule:", Bright(r.NodeID()),
				Bright(durationStr), Green("[OK]"))
		}
		return code, err
	})
}

// Debug is middleware that sets the debug flag to true
func Debug(runner Runner) Runner {
	return RunnerFunc(func(ctx context.Context, r *Rule, opts RunOpts) (Code, error) {
		opts.Debug = true
		return runner.Run(ctx, r, opts)
	})
}
