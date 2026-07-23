package storage

import (
	"context"
	"encoding/json"
	"time"
)

// ReleaseSnapshot holds pre-extracted metadata for a single release,
// stored when a release enters the Ready phase so that future diff
// requests can use the data without re-running oc
type ReleaseSnapshot struct {
	PullSpec    string            `json:"pullSpec"`
	Manifest    json.RawMessage   `json:"manifest"`
	RPMPackages map[string]string `json:"rpmPackages,omitempty"`
	ExtractedAt time.Time         `json:"extractedAt"`
}

// ReleaseSnapshotStorage defines the interface for storing and retrieving
// per-release snapshots, keyed by pull spec
type ReleaseSnapshotStorage interface {
	// Get retrieves a snapshot for the given pull spec
	Get(ctx context.Context, pullSpec string) (*ReleaseSnapshot, bool, error)

	// Set stores a snapshot for the given pull spec
	Set(ctx context.Context, pullSpec string, snapshot *ReleaseSnapshot) error

	// Close cleanly shuts down the storage backend
	Close() error
}

// ReleaseSnapshotLister is an optional interface that storage backends can
// implement to support cache warming by listing all stored snapshots
type ReleaseSnapshotLister interface {
	List(ctx context.Context) ([]*ReleaseSnapshot, error)
}
