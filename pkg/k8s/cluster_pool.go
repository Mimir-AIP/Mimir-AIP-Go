package k8s

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ClusterConfig holds configuration for a single Kubernetes cluster entry.
type ClusterConfig struct {
	Name            string `yaml:"name"`
	Kubeconfig      string `yaml:"kubeconfig"`      // empty = in-cluster
	Namespace       string `yaml:"namespace"`
	OrchestratorURL string `yaml:"orchestratorURL"` // URL workers on this cluster use to reach the orchestrator
	MaxWorkers      int    `yaml:"maxWorkers"`
	ServiceAccount  string `yaml:"serviceAccount"`
}

// ClusterEntry pairs a ClusterConfig with its initialised k8s Client.
type ClusterEntry struct {
	Config ClusterConfig
	Client *Client
}

// ClusterPool manages a prioritised list of Kubernetes clusters for worker dispatch.
// The primary cluster is always the first entry; additional entries receive overflow.
type ClusterPool struct {
	entries   []ClusterEntry
	authToken string // shared WORKER_AUTH_TOKEN injected into every spawned Job
}

// clusterConfigFile is the on-disk YAML format for the cluster config file.
type clusterConfigFile struct {
	Clusters []ClusterConfig `yaml:"clusters"`
}

// NewClusterPool creates a ClusterPool from a slice of ClusterConfig entries.
// authToken is injected as WORKER_AUTH_TOKEN into every Job spawned by any cluster.
func NewClusterPool(configs []ClusterConfig, authToken string) (*ClusterPool, error) {
	pool := &ClusterPool{authToken: authToken}

	for _, cfg := range configs {
		client, err := NewClientWithKubeconfig(ClientConfig{
			Namespace:          cfg.Namespace,
			OrchestratorURL:    cfg.OrchestratorURL,
			ServiceAccountName: cfg.ServiceAccount,
			WorkerAuthToken:    authToken,
		}, cfg.Kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("cluster %q: %w", cfg.Name, err)
		}
		pool.entries = append(pool.entries, ClusterEntry{Config: cfg, Client: client})
	}

	return pool, nil
}

// LoadClusterPool reads a YAML cluster config file and returns an initialised ClusterPool.
func LoadClusterPool(path string, authToken string) (*ClusterPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cluster config file %q: %w", path, err)
	}

	var f clusterConfigFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse cluster config file %q: %w", path, err)
	}

	return NewClusterPool(f.Clusters, authToken)
}

// ActiveWorkerCounts queries each cluster and returns a map of cluster name -> active worker count.
// Clusters that fail to respond are counted as 0 so dispatch can still proceed.
func (p *ClusterPool) ActiveWorkerCounts() map[string]int {
	counts := make(map[string]int, len(p.entries))
	for _, e := range p.entries {
		n, err := e.Client.GetActiveWorkerCount()
		if err != nil {
			n = 0
		}
		counts[e.Config.Name] = n
	}
	return counts
}

// TotalCapacity returns the sum of MaxWorkers across all cluster entries.
func (p *ClusterPool) TotalCapacity() int {
	total := 0
	for _, e := range p.entries {
		total += e.Config.MaxWorkers
	}
	return total
}

// SelectCluster returns the best ClusterEntry to receive a new worker job.
//
// Strategy (burst / primary-first):
//  1. If preferred is non-empty and that cluster has headroom, use it.
//  2. Otherwise iterate entries in declaration order and return the first with
//     available capacity (counts[name] < entry.MaxWorkers).
//
// Returns nil if all clusters are at capacity.
func (p *ClusterPool) SelectCluster(counts map[string]int, preferred string) *ClusterEntry {
	// Preferred cluster override
	if preferred != "" {
		for i := range p.entries {
			e := &p.entries[i]
			if e.Config.Name == preferred && counts[e.Config.Name] < e.Config.MaxWorkers {
				return e
			}
		}
	}

	// Burst: first cluster with headroom
	for i := range p.entries {
		e := &p.entries[i]
		if counts[e.Config.Name] < e.Config.MaxWorkers {
			return e
		}
	}

	return nil
}

// Len returns the number of cluster entries in the pool.
func (p *ClusterPool) Len() int {
	return len(p.entries)
}
