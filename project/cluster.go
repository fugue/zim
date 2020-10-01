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
package project

import (
	"encoding/json"
	"io/ioutil"
)

// ClusterConfig contains information needed to spawn tasks in ECS
type ClusterConfig struct {
	Cluster         string            `json:"cluster"`
	TaskDefinitions map[string]string `json:"task_definitions"`
	SecurityGroup   string            `json:"security_group"`
	Subnets         []string          `json:"subnets"`
	Bucket          string            `json:"bucket"`
	Queue           string            `json:"queue"`
	Athens          string            `json:"athens"`
}

// ReadClusterConfig reads a JSON configuration file from disk
func ReadClusterConfig(fpath string) (*ClusterConfig, error) {
	data, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	var config ClusterConfig
	if err = json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
