package slave

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/fugue/zim/git"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/fugue/zim/project"
	"github.com/fugue/zim/queue"
	"github.com/fugue/zim/store"
)

// Opts contains options used for running a Slave
type Opts struct {
	SQS   sqsiface.SQSAPI
	Store store.Store
	Queue queue.Queue
	Kind  string
}

// Slave runs one or more builds received over its queue
type Slave struct {
	opts  Opts
	sqs   sqsiface.SQSAPI
	store store.Store
	queue queue.Queue
}

// New returns a build slave
func New(opts Opts) *Slave {
	return &Slave{
		sqs:   opts.SQS,
		opts:  opts,
		queue: opts.Queue,
		store: opts.Store,
	}
}

// Run the slave and process any received messages
func (s *Slave) Run(ctx context.Context, done <-chan bool) error {
	running := true
	for running {
		select {
		case <-done:
			running = false
		case <-ctx.Done():
			running = false
		default:
			var msg project.Message
			ok, err := s.queue.Receive(&msg)
			if ok {
				if msg.Exit {
					running = false
					continue
				}
				output, err := s.Process(ctx, &msg)
				if err != nil {
					msg.Success = false
				} else {
					msg.Success = true
				}
				fmt.Println("Processed", msg.BuildID, msg.Component, msg.Rule, err)
				msg.Output = output
				respQueue := msg.ResponseQueue
				if respQueue != "" {
					respQueue := queue.NewSQS(s.sqs, respQueue)
					if err := respQueue.Send(msg); err != nil {
						fmt.Println("Failed to respond", err)
						continue
					}
				}
			} else if err != nil {
				return err
			} else {
				time.Sleep(time.Second)
			}
		}
	}
	fmt.Println("Slave stopping")
	return nil
}

// Process a request to run a rule
func (s *Slave) Process(ctx context.Context, msg *project.Message) (string, error) {

	// TODO: download dependencies from store

	// The source code S3 key must be specified
	if msg.SourceKey == "" {
		return "", errors.New("Source key unset")
	}

	// Source will be unpacked in this local directory during the build
	tmpDir, err := ioutil.TempDir("", "zim-")
	if err != nil {
		return "", fmt.Errorf("Failed to create tmp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download and unpack the source
	err = git.DownloadExtractArchive(ctx, s.store, tmpDir, msg.SourceKey)
	if err != nil {
		return "", fmt.Errorf("Failed to download source: %s", err)
	}

	// Find the requested component to build in the source tree
	proj, err := project.New(tmpDir)
	if err != nil {
		return "", fmt.Errorf("Failed to create project: %s", err)
	}
	rule, found := proj.Rule(msg.Component, msg.Rule)
	if !found {
		return "", fmt.Errorf("Failed to find rule: %s.%s",
			msg.Component, msg.Rule)
	}

	// Any artifacts will be saved to this output folder in S3
	artifactsKey := fmt.Sprintf("artifacts/%s", msg.BuildID)

	// Create processing chain that captures all build output and uploads
	// artifacts to S3
	var outputBuffer bytes.Buffer
	runner := project.NewChain(
		project.NewCapturedOutput(&outputBuffer),
		project.NewArtifactUploader(s.store, artifactsKey),
		project.Logger,
	).Then(&project.StandardRunner{})

	// Run the rule
	executor := project.NewBashExecutor()
	_, err = runner.Run(ctx, rule, project.RunOpts{Executor: executor})
	output := outputBuffer.String()
	return output, err
}
