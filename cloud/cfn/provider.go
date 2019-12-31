package cfn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/LuminalHQ/zim/cache"
	"github.com/LuminalHQ/zim/project"
)

// CloudFormationError is a simple error type
type CloudFormationError string

func (e CloudFormationError) Error() string { return string(e) }

// StackNotFound indicates a stack doesn't exist that matches the search
const StackNotFound = CloudFormationError("Stack not found")

// Provider implements CloudFormation support for Zim
type Provider struct {
	session *session.Session
	cfn     *cf.CloudFormation
}

// Stack represents a CloudFormation stack as a Zim Resource
type Stack struct {
	p    *Provider
	name string
	arn  string
}

// StackState is the basis for determining a hash of the current Stack state.
type StackState struct {
	ID            string    `json:"id"`
	Changeset     string    `json:"changeset"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
}

// New returns a CloudFormation Provider
func New(sess *session.Session) project.Provider {
	return &Provider{}
}

// Init sets up the AWS session for this Provider
func (p *Provider) Init(opts map[string]interface{}) error {

	region, ok := opts["region"].(string)
	if !ok {
		return errors.New("'region' must be provided for the cloudformation provider")
	}

	retries, _ := opts["retries"].(int)
	if retries == 0 {
		retries = 4
	}

	cfg := aws.NewConfig().WithRegion(region).WithMaxRetries(retries)
	sess, err := session.NewSession(cfg)
	if err != nil {
		return err
	}

	p.session = sess
	p.cfn = cf.New(sess)
	return nil
}

// Name identifies the type name of this Provider
func (p *Provider) Name() string {
	return "cloudformation"
}

// New returns a Stack Resource
func (p *Provider) New(path string) project.Resource {
	return NewStack(p, path)
}

// Match Stacks by name
func (p *Provider) Match(pattern string) (project.Resources, error) {
	stack, err := p.DescribeStack(pattern)
	if err != nil {
		return nil, err
	}
	return project.Resources{p.New(*stack.StackName)}, nil
}

// ListStacks finds all CloudFormation stacks
func (p *Provider) ListStacks() ([]*cf.StackSummary, error) {
	ctx := context.Background()
	var matches []*cf.StackSummary
	err := p.cfn.ListStacksPagesWithContext(ctx, &cf.ListStacksInput{},
		func(out *cf.ListStacksOutput, more bool) bool {
			for _, summary := range out.StackSummaries {
				matches = append(matches, summary)
			}
			return true
		})
	if err != nil {
		return nil, err
	}
	return matches, nil
}

// DescribeStack returns information about the named CloudFormation Stack
func (p *Provider) DescribeStack(stackName string) (*cf.Stack, error) {
	ctx := context.Background()
	resp, err := p.cfn.DescribeStacksWithContext(ctx, &cf.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Stacks) == 0 {
		return nil, StackNotFound
	}
	return resp.Stacks[0], nil
}

// NewStack returns a Stack given the path
func NewStack(provider *Provider, path string) *Stack {
	return &Stack{p: provider, name: path}
}

// OnFilesystem is false for files
func (s *Stack) OnFilesystem() bool {
	return false
}

// Cacheable is false for Stacks since they can be uploaded to a cache
func (s *Stack) Cacheable() bool {
	return false
}

// Name of the Resource
func (s *Stack) Name() string {
	return s.name
}

// Path returns the absolute path to the Stack
func (s *Stack) Path() string {
	return s.name
}

// Exists indicates whether the Stack currently exists
func (s *Stack) Exists() (bool, error) {
	if _, err := s.p.DescribeStack(s.name); err != nil {
		if err == StackNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Hash of Stack contents
func (s *Stack) Hash() (string, error) {
	info, err := s.p.DescribeStack(s.name)
	if err != nil {
		return "", err
	}
	state := StackState{
		ID:            *info.StackId,
		Changeset:     *info.ChangeSetId,
		LastUpdatedAt: *info.LastUpdatedTime,
	}
	js, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("Failed to hash stack state: %s", err)
	}
	return cache.HashString(string(js))
}

// LastModified returns the time when this Stack was last updated
func (s *Stack) LastModified() (time.Time, error) {
	info, err := s.p.DescribeStack(s.name)
	if err != nil {
		return time.Time{}, err
	}
	if info.LastUpdatedTime == nil {
		return time.Time{}, nil
	}
	return *info.LastUpdatedTime, nil
}

// AsFile returns the path to the file
func (s *Stack) AsFile() (string, error) {
	return "", errors.New("unsupported")
}
