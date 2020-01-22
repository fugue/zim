package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
}

type token struct {
	Name  string
	Email string
}

type authHandler struct {
	ddb   dynamodbiface.DynamoDBAPI
	table string
}

func (h *authHandler) HandleRequest(ctx context.Context, request events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error) {

	// Extract token from "Bearer <token>" string
	tokenSlice := strings.Split(request.AuthorizationToken, " ")
	var bearerToken string
	if len(tokenSlice) > 1 {
		bearerToken = tokenSlice[len(tokenSlice)-1]
	}

	// Lookup token in DynamoDB to see if it exists
	token, err := h.getToken(ctx, bearerToken)
	if err != nil {
		// Not found or any other error: unauthorized!
		logger.WithError(err).Info("Unauthorized")
		return events.APIGatewayCustomAuthorizerResponse{}, errors.New("Unauthorized")
	}

	logger.WithFields(logrus.Fields{
		"name":  token.Name,
		"email": token.Email,
	}).Info("OK")

	// Return policy that authorizes the request
	return generatePolicy(token.Name, "Allow", request.MethodArn), nil
}

func (h *authHandler) getToken(ctx context.Context, id string) (*token, error) {
	result, err := h.ddb.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(h.table),
		Key: map[string]*dynamodb.AttributeValue{
			"token": {S: aws.String(id)},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return nil, fmt.Errorf("Not found: %s", id)
	}
	var t token
	if err := dynamodbattribute.UnmarshalMap(result.Item, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func generatePolicy(principalID, effect, resource string) events.APIGatewayCustomAuthorizerResponse {
	authResponse := events.APIGatewayCustomAuthorizerResponse{PrincipalID: principalID}
	if effect != "" && resource != "" {
		authResponse.PolicyDocument = events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   effect,
					Resource: []string{resource},
				},
			},
		}
	}
	return authResponse
}

func main() {

	logger.Info("coldstart")

	sess := session.Must(session.NewSession())
	svc := dynamodb.New(sess)

	tableName := os.Getenv("TABLE")
	if tableName == "" {
		logger.Fatal("TABLE is not set")
	}

	handler := &authHandler{
		ddb:   svc,
		table: tableName,
	}
	lambda.Start(handler.HandleRequest)
}
