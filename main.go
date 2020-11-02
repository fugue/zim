// Copyright 2020 Fugue, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

//go:generate mockgen -source=project/runner.go -package project -destination project/runner_mock.go
//go:generate mockgen -source=project/exec.go -package project -destination project/exec_mock.go

import (
	"log"
	"os"
	"runtime/pprof"

	"github.com/fugue/zim/cmd"
)

func main() {

	if os.Getenv("ZIM_PROFILE") == "1" {
		f, err := os.Create("cpuprofile.out")
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	cmd.Execute()
}
