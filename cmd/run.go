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
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fugue/zim/cache"
	"github.com/fugue/zim/project"
	"github.com/fugue/zim/sched"
	"github.com/fugue/zim/store"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func closeHandler(cancel context.CancelFunc) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
		fmt.Println(project.Yellow(" Cleaning up before exiting..."))
	}()
}

// NewRunCommand returns a scheduler command
func NewRunCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run rules",
		Run: func(cmd *cobra.Command, args []string) {

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			closeHandler(cancel)

			opts := getZimOptions(cmd, args)

			// If inside a git repo pick the root as the project directory
			if repo, err := getRepository(opts.Directory); err == nil {
				opts.Directory = repo
			}
			absDir, err := filepath.Abs(opts.Directory)
			if err != nil {
				fatal(err)
			}
			opts.Directory = absDir

			if opts.Jobs < 1 {
				opts.Jobs = 1
			}

			var executor project.Executor
			if opts.UseDocker {
				executor = project.NewDockerExecutor(opts.Directory)
			} else {
				executor = project.NewBashExecutor()
			}

			projDef, componentDefs, err := project.Discover(opts.Directory)
			if err != nil {
				fatal(err)
			}

			// Load selected components from the project
			proj, err := project.NewWithOptions(project.Opts{
				Root:          opts.Directory,
				ProjectDef:    projDef,
				ComponentDefs: componentDefs,
				Executor:      executor,
			})
			if err != nil {
				fatal(err)
			}

			components, err := proj.Select(opts.Components, opts.Kinds)
			if err != nil {
				fatal(err)
			}
			buildID := project.UUID()

			// Create list of middleware to use
			var builders []project.RunnerBuilder
			if opts.Debug {
				builders = append(builders, project.Debug)
			}
			if opts.OutputMode == "buffered" {
				builders = append(builders, project.BufferedOutput)
			}
			builders = append(builders, project.Logger)

			// Add caching middleware depending on configuration
			if opts.CacheMode == cache.Disabled {
				fmt.Fprintf(os.Stdout, project.Yellow("Caching is disabled.\n"))
			} else if opts.URL != "" {
				objStore := store.NewHTTP(opts.URL, opts.Token)
				self, err := user.Current()
				if err != nil {
					fatal(err)
				}
				builders = append(builders,
					cache.NewMiddleware(objStore, self.Name, opts.CacheMode))
			} else {
				fmt.Fprintf(os.Stderr,
					project.Yellow("Cache URL is not set. See the docs!\n"))
			}

			// Chain together all middleware
			runner := project.NewChain(builders...).
				Then(&project.StandardRunner{})

			// Run the scheduler which gives rules to workers to execute
			// in order of rule dependencies
			var schedulerErr error
			scheduler := sched.NewGraphScheduler()
			for _, rule := range opts.Rules {
				rules := components.Rules([]string{rule})
				if len(rules) == 0 {
					return
				}
				schedulerErr = scheduler.Run(ctx, sched.Options{
					BuildID:    buildID,
					Rules:      rules,
					Runner:     runner,
					Executor:   executor,
					NumWorkers: opts.Jobs,
				})
				if schedulerErr != nil {
					break
				}
			}

			if schedulerErr != nil {
				if schedulerErr.Error() == "context canceled" {
					// Wait for cleanup before exiting
					time.Sleep(time.Millisecond * 500)
					os.Exit(1)
				} else {
					fatal(schedulerErr)
				}
			}
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			opts := getZimOptions(cmd, args)
			proj, err := getProject(opts.Directory)
			if err != nil {
				fatal(err)
			}
			comps, err := proj.Select(opts.Components, []string{})
			if err != nil {
				fatal(err)
			}
			return comps.FilterRules(opts.Rules), cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().IntP("jobs", "j", 1, "Concurrent jobs")
	viper.BindPFlag("jobs", cmd.Flags().Lookup("jobs"))

	return cmd
}

func init() {
	rootCmd.AddCommand(NewRunCommand())
}
