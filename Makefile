-include local-config.mk

# Variables relevant for the Zim CLI
BINARY = zim
INSTALLED_BINARY = /usr/local/bin/$(BINARY)
SOURCE = $(wildcard *.go) $(wildcard */*.go) $(wildcard cloud/*/*.go)
GO = GO111MODULE=on go
VERSION = $(shell cat VERSION)
GITCOMMIT = $(shell git rev-parse --short HEAD 2> /dev/null || true)
define LDFLAGS
    -X \"github.com/fugue/zim/cmd.Version=$(VERSION)\" \
    -X \"github.com/fugue/zim/cmd.GitCommit=$(GITCOMMIT)\"
endef
CLI_BUILD = $(GO) build -ldflags="$(LDFLAGS) -s -w"

# Variables relating to the Zim CloudFormation stack
STACK_NAME ?= zim
AWS_REGION ?= us-east-2
SIGNER_SOURCE = $(wildcard signer/*.go sign/*.go)
AUTH_SOURCE = $(wildcard auth/*.go)
SIGNER_DIST = signer.zip
AUTH_DIST = auth.zip

API_URL = $(shell aws cloudformation describe-stacks \
	--region $(AWS_REGION) \
	--stack-name $(STACK_NAME) \
	--query 'Stacks[0].Outputs[?OutputKey==`Api`].OutputValue' \
	--output text)

$(BINARY): $(SOURCE)
	$(CLI_BUILD) -v -o $@

$(BINARY)-linux-amd64: $(SOURCE)
	GOOS=linux GOARCH=amd64 $(CLI_BUILD) -o $@

$(BINARY)-darwin-amd64: $(SOURCE)
	GOOS=darwin GOARCH=amd64 $(CLI_BUILD) -o $@

release: $(BINARY)-linux-amd64 $(BINARY)-darwin-amd64

.PHONY: install
install: $(INSTALLED_BINARY)

$(INSTALLED_BINARY): $(BINARY)
	cp $(BINARY) $@

$(SIGNER_DIST): $(SIGNER_SOURCE)
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-s -w" -o signer_lambda ./signer
	zip $@ signer_lambda
	rm signer_lambda

$(AUTH_DIST): $(AUTH_SOURCE)
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-s -w" -o auth_lambda ./auth
	zip $@ auth_lambda
	rm auth_lambda

.PHONY: stack
stack: $(SIGNER_DIST) $(AUTH_DIST)
	sam deploy \
		--guided \
		--stack-name $(STACK_NAME) \
		--no-fail-on-empty-changeset \
		--region $(AWS_REGION)

.PHONY: deploy
deploy: stack
	@echo ""
	@echo "Add this entry to the file ~/.zim.yaml:"
	@echo ""
	@echo "url: $(API_URL)"
	@echo ""

.PHONY: clean
clean:
	rm -f cmp/cmp
	rm -f coverage.out
	rm -f $(BINARY) $(BINARY)-linux-amd64 $(BINARY)-darwin-amd64
	rm -f $(SIGNER_DIST) $(AUTH_DIST)

.PHONY: test
test:
	$(GO) test -cover ./...

.PHONY: coverage
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: mocks
mocks:
	go generate
