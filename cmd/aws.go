package cmd

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

func getSession(region string) (*session.Session, error) {
	cfg := aws.NewConfig().WithRegion(region).WithMaxRetries(8)
	return session.NewSession(cfg)
}
