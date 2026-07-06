package releasecontroller

import (
	"context"
	"fmt"
	"sort"
)

// AllReleaseStreamsResponse represents the response from /api/v1/releasestreams/all
// It's a map of release stream names to their list of releases
// Example:
//
//	{
//	  "4-stable": ["4.21.10", "4.21.9", "4.21.8", ...],
//	  "4-dev-preview": ["4.22.0-ec.5", "4.22.0-ec.4", ...],
//	  "4.14.0-0.nightly": ["4.14.0-0.nightly-2024-01-15-123456", ...]
//	}
type AllReleaseStreamsResponse map[string][]string

// GetAllReleaseStreams fetches all release streams with all their releases from the release-controller
// This calls the /api/v1/releasestreams/all endpoint
// Returns a map where:
//   - Key: release stream name (e.g., "4-stable", "4.14.0-0.nightly", "4-dev-preview")
//   - Value: array of release names in that stream
func (c *client) GetAllReleaseStreams(ctx context.Context) (AllReleaseStreamsResponse, error) {
	var result AllReleaseStreamsResponse
	if err := c.GetJSON(ctx, "/api/v1/releasestreams/all", &result); err != nil {
		return nil, fmt.Errorf("failed to get all release streams: %w", err)
	}
	return result, nil
}

// GetReadyReleaseStreams fetches release streams with releases in the Ready phase
// This calls the /api/v1/releasestreams/ready endpoint
func (c *client) GetReadyReleaseStreams(ctx context.Context) (AllReleaseStreamsResponse, error) {
	var result AllReleaseStreamsResponse
	if err := c.GetJSON(ctx, "/api/v1/releasestreams/ready", &result); err != nil {
		return nil, fmt.Errorf("failed to get ready release streams: %w", err)
	}
	return result, nil
}

// GetReleaseStreamNames returns the names of all available release streams
// Examples: "4-stable", "4-dev-preview", "4.11.0-0.nightly", "4.14.0-0.ci"
func (c *client) GetReleaseStreamNames(ctx context.Context) ([]string, error) {
	streams, err := c.GetAllReleaseStreams(ctx)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(streams))
	for name := range streams {
		names = append(names, name)
	}

	// Sort for consistent ordering
	sort.Strings(names)
	return names, nil
}

// GetReleasesForStream returns all releases for a specific release stream
// Returns an error if the stream doesn't exist
func (c *client) GetReleasesForStream(ctx context.Context, streamName string) ([]string, error) {
	streams, err := c.GetAllReleaseStreams(ctx)
	if err != nil {
		return nil, err
	}

	releases, found := streams[streamName]
	if !found {
		return nil, fmt.Errorf("stream %s not found", streamName)
	}

	return releases, nil
}

// GetLatestReleaseForStream returns the first (most recent) release for a specific stream
// Returns an error if the stream doesn't exist or has no releases
func (c *client) GetLatestReleaseForStream(ctx context.Context, streamName string) (string, error) {
	releases, err := c.GetReleasesForStream(ctx, streamName)
	if err != nil {
		return "", err
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("stream %s has no releases", streamName)
	}

	return releases[0], nil
}

// StreamStats contains statistics about a release stream
type StreamStats struct {
	Name          string
	ReleaseCount  int
	LatestRelease string
}

// ReleaseLocation describes where a release was found
type ReleaseLocation struct {
	Endpoint   *Endpoint // The release-controller endpoint
	StreamName string    // Name of the stream containing the release
}

// GetStreamStats returns statistics for all release streams
func (c *client) GetStreamStats(ctx context.Context) ([]StreamStats, error) {
	streams, err := c.GetAllReleaseStreams(ctx)
	if err != nil {
		return nil, err
	}

	stats := make([]StreamStats, 0, len(streams))
	for name, releases := range streams {
		stat := StreamStats{
			Name:         name,
			ReleaseCount: len(releases),
		}
		if len(releases) > 0 {
			stat.LatestRelease = releases[0]
		}
		stats = append(stats, stat)
	}

	// Sort by name for consistent ordering
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Name < stats[j].Name
	})

	return stats, nil
}

// FindRelease searches for a specific release across all streams in this endpoint
// Returns all locations (stream names) where the release was found
// Returns an empty slice if the release is not found
func (c *client) FindRelease(ctx context.Context, releaseName string) ([]ReleaseLocation, error) {
	streams, err := c.GetAllReleaseStreams(ctx)
	if err != nil {
		return nil, err
	}

	var locations []ReleaseLocation
	endpoint := c.GetEndpoint()

	for streamName, releases := range streams {
		for _, release := range releases {
			if release == releaseName {
				locations = append(locations, ReleaseLocation{
					Endpoint:   endpoint,
					StreamName: streamName,
				})
				break // Found in this stream, no need to continue
			}
		}
	}

	// Sort by stream name for consistent ordering
	sort.Slice(locations, func(i, j int) bool {
		return locations[i].StreamName < locations[j].StreamName
	})

	return locations, nil
}
