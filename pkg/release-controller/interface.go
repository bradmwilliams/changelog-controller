package releasecontroller

import (
	"context"
	"io"
)

// Client defines the interface for interacting with release-controller endpoints
type Client interface {
	// Get performs a GET request to the specified path on the endpoint
	// Returns the response body reader and any error
	Get(ctx context.Context, path string) (io.ReadCloser, error)

	// GetJSON performs a GET request and unmarshals the JSON response into the provided interface
	GetJSON(ctx context.Context, path string, v interface{}) error

	// GetAllReleaseStreams fetches all release streams with their releases from /api/v1/releasestreams/all
	// Returns a map of stream names to their list of releases
	GetAllReleaseStreams(ctx context.Context) (AllReleaseStreamsResponse, error)

	// GetReleaseStreamNames returns the names of all available release streams
	// (e.g., "4-stable", "4.11.0-0.nightly", "4-dev-preview")
	GetReleaseStreamNames(ctx context.Context) ([]string, error)

	// GetReleasesForStream returns all release names for a specific release stream
	GetReleasesForStream(ctx context.Context, streamName string) ([]string, error)

	// GetLatestReleaseForStream returns the first (most recent) release for a specific stream
	GetLatestReleaseForStream(ctx context.Context, streamName string) (string, error)

	// GetStreamStats returns statistics for all release streams
	GetStreamStats(ctx context.Context) ([]StreamStats, error)

	// FindRelease searches for a specific release across all streams in this endpoint
	// Returns all locations (endpoint + stream name) where the release was found
	FindRelease(ctx context.Context, releaseName string) ([]ReleaseLocation, error)

	// GetReleaseInfo fetches detailed information for a specific release
	// streamName is the release stream (e.g., "4-stable")
	// releaseName is the specific release tag (e.g., "4.21.10")
	GetReleaseInfo(ctx context.Context, streamName, releaseName string) (*ReleaseInfo, error)

	// GetReleaseInfoForLocation is a convenience method that fetches release info
	// using a ReleaseLocation obtained from FindRelease
	GetReleaseInfoForLocation(ctx context.Context, location ReleaseLocation, releaseName string) (*ReleaseInfo, error)

	// GetTagsForStream fetches all tags for a specific release stream
	GetTagsForStream(ctx context.Context, streamName string) ([]Tag, error)

	// GetPullSpec finds the pull specification for a specific release name
	// This searches across all streams in this endpoint to find the release
	GetPullSpec(ctx context.Context, releaseName string) (string, error)

	// GetPullSpecForLocation gets the pull specification for a release at a specific location
	GetPullSpecForLocation(ctx context.Context, location ReleaseLocation, releaseName string) (string, error)

	// GetTagForRelease gets the full Tag information for a specific release
	GetTagForRelease(ctx context.Context, releaseName string) (*Tag, error)

	// GetEndpoint returns the endpoint this client is configured for
	GetEndpoint() *Endpoint

	// Close closes the client and releases any resources
	Close() error
}
