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
