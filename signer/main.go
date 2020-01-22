package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
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

type event struct {
	Method        string            `json:"method"`
	Name          string            `json:"name"`
	Metadata      map[string]string `json:"metadata"`
	ContentLength int64             `json:"content_len"`
}

type output struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type eventHandler struct {
	s3        s3iface.S3API
	bucket    string
	prefix    string
	expireMin int
}

func (h *eventHandler) HandleRequest(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	logger.WithField("req", req).Info("request")

	if req.Path == "/sign" {
		return h.HandleSignRequest(ctx, req)
	}
	return events.APIGatewayProxyResponse{Body: "unknown path", StatusCode: 404}, nil
}

func (h *eventHandler) HandleSignRequest(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	principalID, ok := req.RequestContext.Authorizer["principalId"].(string)
	if !ok || principalID == "" {
		return events.APIGatewayProxyResponse{Body: "unknown principal", StatusCode: 401}, nil
	}

	var e event
	if err := json.Unmarshal([]byte(req.Body), &e); err != nil {
		logger.WithError(err).Error("Failed to unmarshal input")
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}
	logger.WithField("event", e).Info("Sign request")

	output, err := h.Sign(ctx, e)
	if err != nil {
		logger.WithError(err).Error("Failed to sign request")
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}
	logger.WithField("response", output).Info("Signed OK")

	resp, err := json.Marshal(output)
	if err != nil {
		logger.WithError(err).Error("Failed to marshal output")
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}
	return events.APIGatewayProxyResponse{Body: string(resp), StatusCode: 200}, nil
}

func (h *eventHandler) Sign(ctx context.Context, e event) (*output, error) {

	if e.Method != "GET" && e.Method != "PUT" {
		return nil, fmt.Errorf("Invalid method: '%s'", e.Method)
	}

	key := filepath.Join(h.prefix, e.Name)

	var s3Request *request.Request
	if e.Method == "GET" {
		s3Request, _ = h.s3.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(h.bucket),
			Key:    aws.String(key),
		})
	} else {
		metadata := map[string]*string{}
		if e.Metadata != nil {
			for k, v := range e.Metadata {
				metadata[k] = aws.String(v)
			}
		}
		s3Request, _ = h.s3.PutObjectRequest(&s3.PutObjectInput{
			Bucket:        aws.String(h.bucket),
			Key:           aws.String(key),
			Metadata:      metadata,
			ContentLength: aws.Int64(e.ContentLength),
		})
	}

	url, signedHeaders, err := s3Request.PresignRequest(5 * time.Minute)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{}
	for k, v := range signedHeaders {
		if len(v) >= 1 {
			headers[k] = v[0]
		}
	}

	logger.WithFields(logrus.Fields{
		"method":  e.Method,
		"bucket":  h.bucket,
		"key":     key,
		"headers": headers,
	}).Info("Signed URL")

	return &output{URL: url, Headers: headers}, nil
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
