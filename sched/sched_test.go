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
	"os"
	"path"
	"testing"

	"github.com/fugue/zim/definitions"
	"github.com/fugue/zim/project"
	"github.com/stretchr/testify/require"
)

func TestScheduler(t *testing.T) {

	ctx := context.Background()

	dir := testDir()
	defer os.RemoveAll(dir)

	widgetDef := &definitions.Component{
		Path: path.Join(dir, "widget"),
		Name: "widget",
		Rules: map[string]definitions.Rule{
			"test": definitions.Rule{},
			"build": definitions.Rule{
				Requires: []definitions.Dependency{
					{Rule: "test"},
				},
			},
		},
	}

	dongleDef := &definitions.Component{
		Path: path.Join(dir, "dongle"),
		Name: "dongle",
		Rules: map[string]definitions.Rule{
			"ignored": definitions.Rule{},
			"build": definitions.Rule{
				Requires: []definitions.Dependency{
					{Component: "widget", Rule: "build"},
				},
			},
		},
	}

	defs := []*definitions.Component{widgetDef, dongleDef}
	p, err := project.NewWithOptions(project.Opts{
		Root:          dir,
		ComponentDefs: defs,
	})
	require.Nil(t, err)
	buildRules := p.Components().Rules([]string{"build"})
	require.Len(t, buildRules, 2)

	widget := p.Components().WithName("widget").First()
	dongle := p.Components().WithName("dongle").First()

	expectedOrder := []*project.Rule{
		widget.MustRule("test"),
		widget.MustRule("build"),
		dongle.MustRule("build"),
	}

	var got []*project.Rule

	runner := project.RunnerFunc(func(ctx context.Context, rule *project.Rule, opts project.RunOpts) (project.Code, error) {
		got = append(got, rule)
		return project.OK, nil
	})

	err = NewGraphScheduler().Run(ctx, Options{
		BuildID:    "234",
		Runner:     runner,
		Rules:      buildRules,
		NumWorkers: 2,
	})
	if err != nil {
		fmt.Println(err.Error())
	}
	require.Nil(t, err)
	require.Equal(t, expectedOrder, got)
}
