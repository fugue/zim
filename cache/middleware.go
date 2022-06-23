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

package cache

import (
	"context"

	"github.com/fugue/zim/project"
)

// NewMiddleware returns caching middleware
func NewMiddleware(c *Cache) project.RunnerBuilder {

	return project.RunnerBuilder(func(runner project.Runner) project.Runner {

		return project.RunnerFunc(func(ctx context.Context, r *project.Rule, opts project.RunOpts) (project.Code, error) {

			// Caching is only applicable for rules that have cacheable
			// outputs. If this is not the case, run the rule normally.
			outputs := r.Outputs()
			if len(outputs) == 0 || !outputs[0].Cacheable() {
				return runner.Run(ctx, r, opts)
			}

			if c.mode != WriteOnly {
				// Download matching outputs from the cache if they exist
				_, err := c.Read(ctx, r)
				if err == nil {
					return project.Cached, nil // Cache hit
				}
				if err != CacheMiss {
					return project.Error, err // Cache error
				}
			}

			// At this point, the outputs were not cached so build the rule
			code, err := runner.Run(ctx, r, opts)

			// Code "OK" indicates the rule was built which means we can
			// store its outputs in the cache
			if code == project.OK {
				if _, err := c.Write(ctx, r); err != nil {
					return project.Error, err
				}
			}
			return code, err
		})
	})
}
