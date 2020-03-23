package project

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/fugue/zim/queue"
)

// FargateRunner runs a rule on a remote container
type FargateRunner struct {
	SourceKey   string
	SQS         sqsiface.SQSAPI
	SlaveQueues map[string]queue.Queue
}

// Run a rule with the provided executor and other options
func (runner *FargateRunner) Run(ctx context.Context, r *Rule, opts RunOpts) (Code, error) {

	// The component determines which ECS task type is used for the rule
	ecsType := "" // r.Component().ECS().Type
	if ecsType == "" {
		return Error, fmt.Errorf("ECS type unset (%s)", r.NodeID())
	}

	// Lookup the SQS queue corresponding to the ECS task type
	slaveQueue, found := runner.SlaveQueues[ecsType]
	if !found {
		return Error, fmt.Errorf("ECS type queue not found: %s", ecsType)
	}
	buildID := opts.BuildID
	if buildID == "" {
		return Error, fmt.Errorf("Build ID unset")
	}
	requestID := UUID()[:4]

	// Create a new SQS queue for receiving responses from the remote builder
	buildQueue, err := queue.CreateSQS(ctx, runner.SQS, fmt.Sprintf("zim-build-%s-%s", buildID, requestID))
	if err != nil {
		return Error, fmt.Errorf("Failed to create queue: %s", err)
	}
	defer buildQueue.Delete()

	// Send a message to trigger the remote build
	if err := slaveQueue.Send(Message{
		BuildID:       buildID,
		Rule:          r.Name(),
		Component:     r.Component().Name(),
		ResponseQueue: buildQueue.Name(),
		SourceKey:     runner.SourceKey,
	}); err != nil {
		return Error, fmt.Errorf("Failed to send via queue: %s", err)
	}

	// Wait for response message which indicates the build finished
	var resp Message
	for {
		ok, err := buildQueue.Receive(&resp)
		if err != nil {
			fmt.Fprintf(opts.Output, "Remote build failed")
			// fmt.Fprintf(opts.Output, "Remote build failed: %s", ")
			return ExecError, err
		} else if ok {
			break
		} else {
			time.Sleep(time.Second)
		}
	}

	// Show output from the remote build
	for _, line := range strings.Split(strings.TrimSpace(resp.Output), "\n") {
		fmt.Fprintln(opts.Output, line)
	}
	return OK, nil
}
