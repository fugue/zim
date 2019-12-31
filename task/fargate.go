package task

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/s3"
)

// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/using_awslogs.html
// awslogs-stream-prefix
// prefix-name/container-name/ecs-task-id
const (
	LogGroupName    = "/aws/ecs/zim"
	LogStreamPrefix = "zim"
)

type ecsRunner struct {
	ecs *ecs.ECS
	s3  *s3.S3
	cfg FargateConfig
}

// TaskDefinition carries information about an ECS task definition
type TaskDefinition struct {
	ARN     arn.ARN
	Name    string
	Version string
}

// FargateConfig defines a Fargate setup
type FargateConfig struct {
	ContainerName  string
	Subnets        []string
	SecurityGroup  string
	Cluster        string
	TaskDefinition string
	AssignPublicIP bool
	Athens         string
}

// NewFargate returns a task runner backended by ECS Fargate
func NewFargate(sess *session.Session, cfg FargateConfig) Runner {
	ecsAPI := ecs.New(sess)
	s3API := s3.New(sess)
	return &ecsRunner{ecs: ecsAPI, s3: s3API, cfg: cfg}
}

func (r *ecsRunner) Run(ctx context.Context, opts Options) (*Task, error) {

	empty := &Task{}

	definition := r.cfg.TaskDefinition
	if opts.Definition != "" {
		definition = opts.Definition
	}

	if r.cfg.Cluster == "" {
		return empty, errors.New("Cluster unset")
	}
	if definition == "" {
		return empty, errors.New("TaskDefinition unset")
	}
	if r.cfg.ContainerName == "" {
		return empty, errors.New("ContainerName unset")
	}
	if len(r.cfg.Subnets) == 0 {
		return empty, errors.New("Subnets unset")
	}
	if r.cfg.SecurityGroup == "" {
		return empty, errors.New("SecurityGroup unset")
	}

	assignPublicIP := ecs.AssignPublicIpDisabled
	if r.cfg.AssignPublicIP {
		assignPublicIP = ecs.AssignPublicIpEnabled
	}

	var subnets []*string
	for _, subnet := range r.cfg.Subnets {
		subnets = append(subnets, aws.String(subnet))
	}
	securityGroups := []*string{aws.String(r.cfg.SecurityGroup)}

	var environment []*ecs.KeyValuePair
	if opts.Environment != nil {
		for k, v := range opts.Environment {
			environment = append(environment, &ecs.KeyValuePair{
				Name:  aws.String(k),
				Value: aws.String(v),
			})
		}
	}
	if r.cfg.Athens != "" {
		environment = append(environment, &ecs.KeyValuePair{
			Name:  aws.String("GOPROXY"),
			Value: aws.String(fmt.Sprintf("http://%s:3000", r.cfg.Athens)),
		})
	}

	containerOverrides := []*ecs.ContainerOverride{
		&ecs.ContainerOverride{
			Name:        aws.String(r.cfg.ContainerName),
			Environment: environment,
		},
	}
	if opts.Memory > 0 {
		containerOverrides[0].Memory = &opts.Memory
	}
	if opts.CPU > 0 {
		containerOverrides[0].Cpu = &opts.CPU
	}

	result, err := r.ecs.RunTaskWithContext(ctx, &ecs.RunTaskInput{
		LaunchType:     aws.String(ecs.LaunchTypeFargate),
		Cluster:        aws.String(r.cfg.Cluster),
		TaskDefinition: aws.String(definition),
		NetworkConfiguration: &ecs.NetworkConfiguration{
			AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
				AssignPublicIp: aws.String(assignPublicIP),
				SecurityGroups: securityGroups,
				Subnets:        subnets,
			},
		},
		Overrides: &ecs.TaskOverride{ContainerOverrides: containerOverrides},
	})
	if err != nil {
		return empty, fmt.Errorf("Failed to run task: %s", err)
	}
	if len(result.Tasks) == 0 {
		return empty, fmt.Errorf("Failed to run task: %s", result.String())
	}
	if len(result.Tasks) != 1 {
		panic(fmt.Sprintf("Unexpected tasks length: %d", len(result.Tasks)))
	}
	task := result.Tasks[0]
	taskID := strings.Split(*task.TaskArn, "/")[1]
	return &Task{
		ID:        taskID,
		ARN:       *task.TaskArn,
		LogGroup:  LogGroupName,
		LogStream: fmt.Sprintf("zim/zim/%s", taskID),
	}, nil
}

func (r *ecsRunner) WaitUntilRunning(ctx context.Context, tasks ...*Task) error {
	if len(tasks) == 0 {
		return nil
	}
	var arns []*string
	for _, task := range tasks {
		arns = append(arns, aws.String(task.ID))
	}
	err := r.ecs.WaitUntilTasksRunningWithContext(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(r.cfg.Cluster),
		Tasks:   arns,
	})
	if err != nil {
		return fmt.Errorf("Failed to wait until tasks running: %s", err)
	}
	return nil
}

func (r *ecsRunner) WaitUntilStopped(ctx context.Context, tasks ...*Task) error {
	if len(tasks) == 0 {
		return nil
	}
	var arns []*string
	for _, task := range tasks {
		arns = append(arns, aws.String(task.ID))
	}
	err := r.ecs.WaitUntilTasksStoppedWithContext(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(r.cfg.Cluster),
		Tasks:   arns,
	})
	if err != nil {
		return fmt.Errorf("Failed to wait until tasks stopped: %s", err)
	}
	return nil
}
