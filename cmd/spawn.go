package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/LuminalHQ/zim/project"
	"github.com/LuminalHQ/zim/task"
	"github.com/go-yaml/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NOTE: NOT USED RIGHT NOW

type fargateConfig struct {
	Cluster         string            `yaml:"cluster"`
	Subnets         []string          `yaml:"subnets"`
	SecurityGroup   string            `yaml:"security_group"`
	TaskDefinitions map[string]string `yaml:"task_definitions"`
	Queues          map[string]string `yaml:"queues"`
	Bucket          string            `yaml:"bucket"`
	Athens          string            `yaml:"athens"`
}

func loadFargateConfig() (*fargateConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configPath := path.Join(home, ".zim", "config.json")
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg fargateConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// NewSpawnCommand returns a command that creates new build slaves
func NewSpawnCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "spawn",
		Short: "Spawn slaves that accepts build jobs over a message queue",
		Run: func(cmd *cobra.Command, args []string) {

			cfg, err := loadFargateConfig()
			if err != nil {
				fatal(fmt.Errorf("Unable to read ~/.zim/config.json: %s", err))
			}

			opts := getZimOptions()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			sess, _ := awsInit(opts)

			runner := task.NewFargate(sess, task.FargateConfig{
				ContainerName: "zim",
				Subnets:       cfg.Subnets,
				SecurityGroup: cfg.SecurityGroup,
				Cluster:       cfg.Cluster,
				Athens:        cfg.Athens,
			})

			count := viper.GetInt("count")
			timeout := viper.GetInt("timeout")
			types := viper.GetStringSlice("types")

			var tasks []*task.Task
			start := time.Now()

			for _, typ := range types {
				typDef, found := cfg.TaskDefinitions[typ]
				if !found {
					fatal(fmt.Errorf("No task definition for %s", typ))
				}
				queue, found := cfg.Queues[typ]
				if !found {
					fatal(fmt.Errorf("No queue for %s", typ))
				}
				for i := 0; i < count; i++ {
					workerID := project.UUID()
					taskOpt := task.Options{
						// Kind:   kind,
						// CPU:    cpu,
						// Memory: memory,
						Definition: typDef,
						Environment: map[string]string{
							"ZIM_BUCKET":  opts.Bucket,
							"ZIM_REGION":  opts.Region,
							"ZIM_STORE":   "1",
							"ZIM_CACHE":   "cache",
							"ZIM_QUEUE":   queue,
							"ZIM_SLAVE":   workerID,
							"ZIM_TIMEOUT": strconv.Itoa(timeout),
						},
					}
					task, err := runner.Run(ctx, taskOpt)
					if err != nil {
						fatal(err)
					}
					fmt.Println("Task:", task.ARN)
					tasks = append(tasks, task)
				}
			}

			fmt.Println("Waiting for tasks to start...")
			if err := runner.WaitUntilRunning(ctx, tasks...); err != nil {
				fatal(err)
			}

			dt := time.Now().Sub(start)
			fmt.Println(len(tasks), "tasks running after", dt)
		},
	}

	var slaveCount int
	var slaveTimeout int
	var taskTypes []string

	cmd.Flags().IntVar(&slaveCount, "count", 2, "Number of slaves to create")
	cmd.Flags().IntVar(&slaveTimeout, "timeout", 600, "Slave timeout (sec)")
	cmd.Flags().StringSliceVar(&taskTypes, "types", []string{"go"}, "Container types")

	viper.BindPFlag("count", cmd.Flags().Lookup("count"))
	viper.BindPFlag("timeout", cmd.Flags().Lookup("timeout"))
	viper.BindPFlag("types", cmd.Flags().Lookup("types"))

	return cmd
}

// func init() {
// 	rootCmd.AddCommand(NewSpawnCommand())
// }
