package main

//go:generate mockgen -source=project/runner.go -package project -destination project/runner_mock.go
//go:generate mockgen -source=project/exec.go -package project -destination project/exec_mock.go

import "github.com/fugue/zim/cmd"

func main() {
	cmd.Execute()
}
