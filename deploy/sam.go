package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Sam is an AWS SAM app that implements the Deployer interface
type Sam struct {
	Bucket string
}

// NewSamDeployer returns a Deployer using AWS SAM
func NewSamDeployer(bucket string) Deployer {
	return &Sam{Bucket: bucket}
}

// Deploy the application
func (sam *Sam) Deploy(ctx context.Context, c *Component, opts DeployOpts) (Deployment, error) {

	deployment := Deployment{StartedAt: time.Now()}
	defer func() {
		deployment.FinishedAt = time.Now()
	}()

	if err := sam.pack(ctx, sam.Bucket, c); err != nil {
		return deployment, err
	}

	var params []string
	for k, v := range opts.Parameters {
		params = append(params, fmt.Sprintf("%s=%s", k, v))
	}

	err := sam.deploy(ctx, c, opts.EnvType, opts.Name, params)
	if err != nil {
		return deployment, err
	}
	return deployment, nil
}

func (sam *Sam) pack(ctx context.Context, bucket string, c *Component) error {
	args := []string{
		"package",
		"--template-file",
		"cloudformation.yaml",
		"--output-template-file",
		"cloudformation_deploy.yaml",
		"--s3-bucket",
		bucket,
	}
	command := exec.CommandContext(ctx, "sam", args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Dir = c.Directory()
	return command.Run()
}

func (sam *Sam) deploy(ctx context.Context, c *Component, envType, stackName string, params []string) error {
	args := []string{
		"deploy",
		"--template-file",
		"cloudformation_deploy.yaml",
		"--stack-name",
		stackName,
		"--capabilities",
		"CAPABILITY_NAMED_IAM",
		"--no-fail-on-empty-changeset",
		"--parameter-overrides",
		fmt.Sprintf("EnvType=%s", envType),
	}
	for _, p := range params {
		args = append(args, p)
	}
	command := exec.CommandContext(ctx, "sam", args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Dir = c.Directory()
	return command.Run()
}
