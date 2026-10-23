package storage

import "context"

// ChangelogStorage defines the interface for storing and retrieving changelog results
type ChangelogStorage interface {
	// Get retrieves a changelog for the given version range
	// Returns the changelog content and true if found, empty string and false otherwise
	Get(ctx context.Context, from, to string) (string, bool, error)

	// Set stores a changelog for the given version range
	Set(ctx context.Context, from, to, changelog string) error

	// Close cleanly shuts down the storage backend
	Close() error
}

// ChangelogEntry represents a single changelog entry with version information
type ChangelogEntry struct {
	FromVersion string
	ToVersion   string
	Changelog   string
}

// ChangelogLister is an optional interface that storage backends can implement
// to support cache warming by listing all stored changelogs
type ChangelogLister interface {
	// List returns all changelog entries stored in the backend
	// This is used for cache warming on startup
	List(ctx context.Context) ([]ChangelogEntry, error)
}

// ChangelogCounter is an optional interface that storage backends can implement
// to report their entry count without fetching changelog contents
type ChangelogCounter interface {
	// Count returns the number of changelog entries stored in the backend
	Count(ctx context.Context) (int, error)
}
