package deploy

import (
	"context"
	"time"
)

// Config stores configuration
type Config struct {
	config map[string]interface{}
}

// Get a config value
func (c *Config) Get(key string) (interface{}, bool) {
	value, found := c.config[key]
	return value, found
}

// GetString returns the configuration value for the given key
func (c *Config) GetString(key string) string {
	value, found := c.Get(key)
	if !found {
		return ""
	}
	strVal, ok := value.(string)
	if !ok {
		return ""
	}
	return strVal
}

// Set a config value
func (c *Config) Set(key string, value interface{}) {
	c.config[key] = value
}

// DeployOpts configure a deploy action
type DeployOpts struct {
	Name       string            `json:"name"`
	EnvType    string            `json:"env_type"`
	Config     Config            `json:"config"`
	Parameters map[string]string `json:"parameters"`
}

// Deployment contains information about the result of a deploy
type Deployment struct {
	StartedAt  time.Time
	FinishedAt time.Time
	Status     string
	Error      error
	Log        string
}

// Deployer is an interface used to Deploy a Component
type Deployer interface {
	Deploy(context.Context, *Component, DeployOpts) (Deployment, error)
}
