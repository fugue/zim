package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fugue/zim/sign"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
}

type eventHandler struct {
	s3        s3iface.S3API
	bucket    string
	prefix    string
	expireMin int
}

func (h *eventHandler) HandleRequest(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	logger.WithField("req", req).Info("request")

	principalID, ok := req.RequestContext.Authorizer["principalId"].(string)
	if !ok || principalID == "" {
		return events.APIGatewayProxyResponse{Body: "unknown principal", StatusCode: 401}, nil
	}

	var input sign.Input
	if err := json.Unmarshal([]byte(req.Body), &input); err != nil {
		logger.WithError(err).Error("Failed to unmarshal input")
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}

	var err error
	var output interface{}

	if req.Path == "/sign" {
		output, err = h.Sign(ctx, &input)
	} else if req.Path == "/head" {
		output, err = h.Head(ctx, &input)
	} else {
		return events.APIGatewayProxyResponse{Body: "unknown path", StatusCode: 404}, nil
	}

	if err != nil {
		logger.WithError(err).Info("Error handling request")
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}

	js, err := json.Marshal(output)
	if err != nil {
		logger.WithError(err).Info("Error marshaling repsonse")
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}

	logger.WithFields(logrus.Fields{
		"principal": principalID,
		"resp":      string(js),
		"err":       err,
	}).Info("OK")

	return events.APIGatewayProxyResponse{Body: string(js), StatusCode: 200}, nil
}

func (h *eventHandler) Sign(ctx context.Context, input *sign.Input) (*sign.Output, error) {

	if input.Method != "GET" && input.Method != "PUT" {
		return nil, fmt.Errorf("Invalid method: '%s'", input.Method)
	}

	key := filepath.Join(h.prefix, input.Name)

	var s3Request *request.Request
	if input.Method == "GET" {
		s3Request, _ = h.s3.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(h.bucket),
			Key:    aws.String(key),
		})
	} else {
		metadata := map[string]*string{}
		if input.Metadata != nil {
			for k, v := range input.Metadata {
				metadata[k] = aws.String(v)
			}
		}
		logger.WithFields(logrus.Fields{
			"bucket": h.bucket,
			"key":    key,
			"meta":   metadata,
		}).Info("PutObjectRequest")
		s3Request, _ = h.s3.PutObjectRequest(&s3.PutObjectInput{
			Bucket:   aws.String(h.bucket),
			Key:      aws.String(key),
			Metadata: metadata,
			// ContentLength: aws.Int64(input.ContentLength),
		})
	}

	url, signedHeaders, err := s3Request.PresignRequest(5 * time.Minute)
	if err != nil {
		logger.WithError(err).Info("Failed to sign request")
		return nil, err
	}

	headers := map[string]string{}
	for k, v := range signedHeaders {
		if len(v) >= 1 {
			headers[k] = v[0]
		}
	}

	logger.WithFields(logrus.Fields{
		"method":  input.Method,
		"bucket":  h.bucket,
		"key":     key,
		"headers": headers,
	}).Info("Signed URL")

	return &sign.Output{URL: url, Headers: headers}, nil
}

func (h *eventHandler) Head(ctx context.Context, input *sign.Input) (*sign.Item, error) {
	key := filepath.Join(h.prefix, input.Name)
	head, err := h.s3.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(h.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return &sign.Item{Key: key}, nil
		}
		return nil, fmt.Errorf("Head failed %s: %s", key, err)
	}
	item := &sign.Item{
		Key:      key,
		Metadata: map[string]string{},
	}
	if head.VersionId != nil {
		item.Version = *head.VersionId
	}
	if head.ETag != nil {
		item.ETag = *head.ETag
	}
	if head.ContentLength != nil {
		item.Size = *head.ContentLength
	}
	if head.LastModified != nil {
		item.LastModified = *head.LastModified
	}
	if head.Metadata != nil {
		for k, v := range head.Metadata {
			item.Metadata[k] = *v
		}
	}
	return item, nil
}

func main() {

	logger.Info("coldstart")

	sess := session.Must(session.NewSession())
	svc := s3.New(sess)

	bucketName := os.Getenv("BUCKET")
	bucketPrefix := os.Getenv("BUCKET_PREFIX")
	expireStr := os.Getenv("EXPIRE_MINUTES")

	if bucketName == "" {
		logger.Fatal("BUCKET is not set")
	}
	if expireStr == "" {
		expireStr = "5"
	}
	expireMin, err := strconv.Atoi(expireStr)
	if err != nil {
		logger.WithError(err).Fatal("Invalid EXPIRE_MINUTES")
	}

	handler := &eventHandler{
		s3:        svc,
		bucket:    bucketName,
		prefix:    bucketPrefix,
		expireMin: expireMin,
	}
	lambda.Start(handler.HandleRequest)
}

func isNotFound(err error) bool {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case "NotFound":
			return true
		}
	}
	return false
}
