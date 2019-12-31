package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

// Queue used to send and receive Messages
type Queue interface {

	// Send a message on the Queue
	Send(interface{}) error

	// Receive a message from the Queue
	Receive(interface{}) (bool, error)

	// Name of the queue
	Name() string

	// Delete the queue
	Delete() error
}

// NewSQS returns a Queue backed by SQS
func NewSQS(api sqsiface.SQSAPI, name string) Queue {
	return &sqsQueue{name: name, api: api}
}

// CreateSQS returns a newly created SQS queue
func CreateSQS(ctx context.Context, api sqsiface.SQSAPI, name string) (Queue, error) {
	resp, err := api.CreateQueueWithContext(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(name),
	})
	if err != nil {
		return nil, err
	}
	url := *resp.QueueUrl
	return NewSQS(api, url), nil
}

type sqsQueue struct {
	name string
	api  sqsiface.SQSAPI
}

// Delete the queue
func (q *sqsQueue) Delete() error {
	_, err := q.api.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(q.name),
	})
	return err
}

// Name returns the Queue name
func (q *sqsQueue) Name() string {
	return q.name
}

// Send a message on the Queue
func (q *sqsQueue) Send(obj interface{}) error {
	js, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("Failed to marshal message: %s", err)
	}
	// mID := uuid.NewV4().String()
	_, err = q.api.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: aws.Int64(0),
		MessageBody:  aws.String(string(js)),
		QueueUrl:     &q.name,
		// MessageGroupId: aws.String("zim"),
		// MessageDeduplicationId: aws.String(mID),
	})
	if err != nil {
		return fmt.Errorf("Failed to send on queue %s: %s", q.name, err)
	}
	return nil
}

// Receive a message from the Queue
func (q *sqsQueue) Receive(obj interface{}) (bool, error) {

	result, err := q.api.ReceiveMessage(&sqs.ReceiveMessageInput{
		AttributeNames: []*string{
			aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		QueueUrl:            &q.name,
		MaxNumberOfMessages: aws.Int64(1),
		VisibilityTimeout:   aws.Int64(20),
		WaitTimeSeconds:     aws.Int64(1),
	})
	if err != nil {
		return false, fmt.Errorf("Failed to receive from queue %s: %s", q.name, err)
	}
	if len(result.Messages) == 0 {
		return false, nil
	}
	message := result.Messages[0]

	_, err = q.api.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &q.name,
		ReceiptHandle: message.ReceiptHandle,
	})
	if err != nil {
		return false, err
	}
	if message.Body == nil {
		return true, fmt.Errorf("Empty message body from queue %s: %s", q.name, err)
	}
	if err := json.Unmarshal([]byte(*message.Body), &obj); err != nil {
		return true, fmt.Errorf("Failed to unmarshal message %s: %s", q.name, err)
	}
	return true, nil
}
