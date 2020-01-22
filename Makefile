-include local-config.mk

# Variables relevant for the Zim CLI
BINARY = zim
INSTALLED_BINARY = /usr/local/bin/$(BINARY)
SOURCE = $(wildcard *.go) $(wildcard */*.go) $(wildcard cloud/*/*.go)
GO = GO111MODULE=on go

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
	$(GO) build -v -o $(BINARY)

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

.PHONY: deploy
deploy: $(SIGNER_DIST) $(AUTH_DIST)
	# --guided
	sam deploy \
		--stack-name $(STACK_NAME) \
		--no-fail-on-empty-changeset \
		--region $(AWS_REGION)
	@echo $(API_URL)

.PHONY: clean
clean:
	rm -f cmp/cmp
	rm -f coverage.out
	rm -f $(BINARY)
	rm -f $(SIGNER_DIST) $(AUTH_DIST)

.PHONY: test
test:
	$(GO) test -v -cover ./project ./graph ./sched ./git ./cache ./format

.PHONY: coverage
coverage:
	go test ./project ./graph ./sched ./git -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: mocks
mocks:
	go generate
