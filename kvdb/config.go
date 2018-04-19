package kvdb

import "fmt"

const (
	KvRootPath = "swallow"
)

type Config struct {
	Type      string   `json:"type"`
	EndPoints []string `json:"endpoints"`
	ClusterID string   `json:"clusterID"`
	LeaderID  string   `json:"leaderID,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		Type:      "zookeeper",
		EndPoints: []string{"localhost:2181", "localhost:2182", "localhost:2183"},
		ClusterID: "defaultZk",
		LeaderID:  "localhost:8888",
	}
}

func (c *Config) getLeaderID() string {
	return c.LeaderID
}

func (c *Config) getRootNode() string {
	return fmt.Sprintf("/%s", KvRootPath)
}

func (c *Config) getProjectNode() string {
	return fmt.Sprintf("/%s/%s", KvRootPath, c.ClusterID)
}

func (c *Config) getElectNode() string {
	return fmt.Sprintf("/%s/%s/election", KvRootPath, c.ClusterID)
}

func (c *Config) getNodeSpecPath() string {
	return fmt.Sprintf("/%s/%s/nodes", KvRootPath, c.ClusterID)
}

func (c *Config) getTaskSpecPath() string {
	return fmt.Sprintf("/%s/%s/tasks", KvRootPath, c.ClusterID)
}
