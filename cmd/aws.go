package cmd

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/LuminalHQ/zim/project"
	"github.com/LuminalHQ/zim/queue"
	"github.com/LuminalHQ/zim/store"
	"github.com/LuminalHQ/zim/task"
)

func getSession(region string) (*session.Session, error) {
	cfg := aws.NewConfig().WithRegion(region).WithMaxRetries(8)
	return session.NewSession(cfg)
}

func getStore(sess *session.Session, bucket string) store.Store {
	return store.NewS3(s3.New(sess), bucket)
}

func getQueue(sess *session.Session, name string) queue.Queue {
	return queue.NewSQS(sqs.New(sess), name)
}

func newQueue(sess *session.Session, name string) (queue.Queue, string, error) {
	svc := sqs.New(sess)
	resp, err := svc.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(name),
	})
	if err != nil {
		return nil, "", err
	}
	url := *resp.QueueUrl
	return queue.NewSQS(svc, url), url, nil
}

func awsInit(opts zimOptions) (sess *session.Session, s store.Store) {

	var err error
	sess, err = getSession(opts.Region)
	session.Must(sess, err)

	if opts.Bucket != "" {
		s = getStore(sess, opts.Bucket)
	}
	return
}

func getTaskRunner(sess *session.Session) task.Runner {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fatal(err)
	}

	cfgPath := path.Join(homeDir, ".zim", "aws.json")
	cfg, err := project.ReadClusterConfig(cfgPath)
	if err != nil {
		fatal(err)
	}

	return task.NewFargate(sess, task.FargateConfig{
		ContainerName: "zim",
		// TaskDefinition: cfg.TaskDefinition,
		Subnets:        cfg.Subnets,
		SecurityGroup:  cfg.SecurityGroup,
		Cluster:        cfg.Cluster,
		AssignPublicIP: false,
		Athens:         cfg.Athens,
	})
}

func listTaskDefinitions(sess *session.Session) (defs []task.TaskDefinition, finalErr error) {

	ctx := context.Background()
	svc := ecs.New(sess)

	input := &ecs.ListTaskDefinitionsInput{}
	svc.ListTaskDefinitionsPagesWithContext(ctx, input,
		func(page *ecs.ListTaskDefinitionsOutput, done bool) bool {
			for _, arnPtr := range page.TaskDefinitionArns {
				taskARN, err := arn.Parse(*arnPtr)
				if err != nil {
					finalErr = err
					return false
				}
				parts := strings.Split(taskARN.Resource, "/")
				if len(parts) != 2 {
					finalErr = fmt.Errorf("Unexpected resource fmt: %s", taskARN.Resource)
					return false
				}
				nameParts := strings.Split(parts[1], ":")
				if len(nameParts) != 2 {
					finalErr = fmt.Errorf("Unexpected resource fmt: %s", taskARN.Resource)
					return false
				}
				defs = append(defs, task.TaskDefinition{
					ARN:     taskARN,
					Name:    nameParts[0],
					Version: nameParts[1],
				})
			}
			return true
		})

	return
}

func mapTaskDefinitions(defs []task.TaskDefinition) map[string]task.TaskDefinition {
	res := map[string]task.TaskDefinition{}
	for _, def := range defs {
		res[def.Name] = def
	}
	return res
}
