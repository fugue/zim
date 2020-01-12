-include local-config.mk

AWS_DEFAULT_REGION = us-east-2
BINARY = zim
INSTALLED_BINARY = /usr/local/bin/$(BINARY)
LINUX_BINARY = zim_linux
SOURCE = $(wildcard *.go) $(wildcard */*.go) $(wildcard cloud/*/*.go)
GO = GO111MODULE=on go
ZIM_CONFIG_DIR = $$HOME/.zim
ZIM_CONFIG = $(ZIM_CONFIG_DIR)/config.json

IMAGE_GO = curtisfugue/golang
IMAGE_PYTHON = curtisfugue/python
IMAGE_NODE = curtisfugue/node
IMAGE_HASKELL = curtisfugue/haskell

$(BINARY): $(SOURCE)
	$(GO) build -v -o $(BINARY)

.PHONY: install
install: $(INSTALLED_BINARY)

$(INSTALLED_BINARY): $(BINARY)
	cp $(BINARY) $@

$(LINUX_BINARY): $(SOURCE)
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-s -w" -o zim_linux

.PHONY: deploy
deploy: $(ZIM_CONFIG)

$(ZIM_CONFIG_DIR):
	mkdir -p $(ZIM_CONFIG_DIR)

$(ZIM_CONFIG): $(ZIM_CONFIG_DIR)
	aws cloudformation deploy \
		--region $(AWS_DEFAULT_REGION) \
		--template-file fargate/cfn.yaml \
		--capabilities CAPABILITY_NAMED_IAM \
		--no-fail-on-empty-changeset \
		--stack-name zim \
		--parameter-overrides \
			ServiceName=zim \
			StackName=zim \
			GoImage=$(IMAGE_GO) \
			PythonImage=$(IMAGE_PYTHON) \
			NodeImage=$(IMAGE_NODE) \
			HaskellImage=$(IMAGE_HASKELL) \
			ContainerCpu=2048 \
			ContainerMemory=8192 \
			BucketUsers=$(BUCKET_USERS)
	AWS_DEFAULT_REGION=$(AWS_DEFAULT_REGION) fargate/config.sh > $@

.PHONY: docker_images
docker_images: $(LINUX_BINARY)
	docker build -t $(IMAGE_GO) -f fargate/Dockerfile.go .
	docker build -t $(IMAGE_PYTHON) -f fargate/Dockerfile.python .
	docker build -t $(IMAGE_NODE) -f fargate/Dockerfile.node .
	docker build -t $(IMAGE_HASKELL) -f fargate/Dockerfile.haskell .

.PHONY: clean
clean:
	rm -f cmp/cmp
	rm -f coverage.out
	rm -f $(BINARY)
	rm -f $(LINUX_BINARY)
	rm -f fargate/config.json
	docker rmi -f $(IMAGE_GO) $(IMAGE_PYTHON) $(IMAGE_NODE) $(IMAGE_HASKELL)

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
