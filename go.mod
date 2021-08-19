module github.com/fugue/zim

go 1.12

require (
	github.com/aws/aws-lambda-go v1.13.3
	github.com/aws/aws-sdk-go v1.25.25
	github.com/bmatcuk/doublestar v1.1.5
	github.com/containerd/containerd v1.5.5 // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/fatih/color v1.7.0
	github.com/fatih/structs v1.1.0
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/golang/mock v1.4.1
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/hashicorp/go-multierror v1.0.0
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.0.1-0.20200710201246-675ae5f5a98c
	github.com/spf13/viper v1.5.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/net v0.0.0-20210226172049-e18ecbb05110
	gonum.org/v1/gonum v0.6.0
)

replace github.com/docker/docker => github.com/docker/engine v0.0.0-20191113042239-ea84732a7725
