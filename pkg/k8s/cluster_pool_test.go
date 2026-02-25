package k8s

import (
	"testing"
)

// mockClusterPool creates a ClusterPool with pre-built entries and no real k8s clients,
// allowing SelectCluster and TotalCapacity to be unit-tested without a live cluster.
func mockPool(entries []ClusterEntry) *ClusterPool {
	return &ClusterPool{entries: entries}
}

func entry(name string, maxWorkers int) ClusterEntry {
	return ClusterEntry{
		Config: ClusterConfig{Name: name, MaxWorkers: maxWorkers},
		Client: nil, // not used by SelectCluster / TotalCapacity
	}
}

// TestSelectCluster_BurstOrder verifies that the first cluster with headroom is chosen.
func TestSelectCluster_BurstOrder(t *testing.T) {
	pool := mockPool([]ClusterEntry{
		entry("primary", 2),
		entry("site-b", 5),
	})

	// primary full, site-b has headroom
	counts := map[string]int{"primary": 2, "site-b": 3}
	got := pool.SelectCluster(counts, "")
	if got == nil {
		t.Fatal("expected a cluster entry, got nil")
	}
	if got.Config.Name != "site-b" {
		t.Errorf("expected site-b, got %q", got.Config.Name)
	}
}

// TestSelectCluster_PrimaryFirst verifies that the primary cluster is preferred when it has headroom.
func TestSelectCluster_PrimaryFirst(t *testing.T) {
	pool := mockPool([]ClusterEntry{
		entry("primary", 5),
		entry("site-b", 5),
	})

	counts := map[string]int{"primary": 2, "site-b": 0}
	got := pool.SelectCluster(counts, "")
	if got == nil {
		t.Fatal("expected a cluster entry, got nil")
	}
	if got.Config.Name != "primary" {
		t.Errorf("expected primary (first with headroom), got %q", got.Config.Name)
	}
}

// TestSelectCluster_PreferredCluster verifies that an explicit preferred cluster is honoured
// when it has available capacity.
func TestSelectCluster_PreferredCluster(t *testing.T) {
	pool := mockPool([]ClusterEntry{
		entry("primary", 10),
		entry("gpu-cluster", 4),
	})

	counts := map[string]int{"primary": 0, "gpu-cluster": 2}
	got := pool.SelectCluster(counts, "gpu-cluster")
	if got == nil {
		t.Fatal("expected a cluster entry, got nil")
	}
	if got.Config.Name != "gpu-cluster" {
		t.Errorf("expected gpu-cluster, got %q", got.Config.Name)
	}
}

// TestSelectCluster_PreferredFull falls back to burst order when the preferred cluster is at capacity.
func TestSelectCluster_PreferredFull(t *testing.T) {
	pool := mockPool([]ClusterEntry{
		entry("primary", 5),
		entry("gpu-cluster", 2),
	})

	// gpu-cluster is full; should fall back to primary
	counts := map[string]int{"primary": 1, "gpu-cluster": 2}
	got := pool.SelectCluster(counts, "gpu-cluster")
	if got == nil {
		t.Fatal("expected a cluster entry, got nil")
	}
	if got.Config.Name != "primary" {
		t.Errorf("expected primary (fallback burst), got %q", got.Config.Name)
	}
}

// TestSelectCluster_AllFull returns nil when every cluster is at capacity.
func TestSelectCluster_AllFull(t *testing.T) {
	pool := mockPool([]ClusterEntry{
		entry("primary", 2),
		entry("site-b", 3),
	})

	counts := map[string]int{"primary": 2, "site-b": 3}
	got := pool.SelectCluster(counts, "")
	if got != nil {
		t.Errorf("expected nil when all clusters full, got %q", got.Config.Name)
	}
}

// TestTotalCapacity verifies the sum of all cluster maxWorkers.
func TestTotalCapacity(t *testing.T) {
	pool := mockPool([]ClusterEntry{
		entry("primary", 10),
		entry("site-b", 20),
		entry("cloud", 5),
	})

	if cap := pool.TotalCapacity(); cap != 35 {
		t.Errorf("expected total capacity 35, got %d", cap)
	}
}
