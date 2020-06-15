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

	"github.com/fatih/color"
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

			opts := getZimOptions()

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

			// Rules can be specified by arguments or options
			if len(opts.Rules) == 0 && len(args) > 0 {
				opts.Rules = args
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

			var objStore store.Store
			if opts.URL != "" {
				objStore = store.NewHTTP(opts.URL, opts.Token)
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

			// Construct build middleware
			var builders []project.RunnerBuilder
			if opts.Debug {
				builders = append(builders, project.Debug)
			}
			builders = append(builders,
				project.BufferedOutput,
				project.Logger,
			)
			if objStore != nil {
				self, err := user.Current()
				if err != nil {
					fatal(err)
				}
				builders = append(builders, cache.NewMiddleware(
					objStore, self.Name, opts.CacheMode))
			} else {
				yellow := color.New(color.FgYellow).SprintFunc()
				fmt.Fprintf(os.Stderr, yellow("Caching is not enabled. See the docs!\n"))
			}
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
	}

	cmd.Flags().IntP("jobs", "j", 1, "Concurrent jobs")
	viper.BindPFlag("jobs", cmd.Flags().Lookup("jobs"))

	return cmd
}

func init() {
	rootCmd.AddCommand(NewRunCommand())
}
