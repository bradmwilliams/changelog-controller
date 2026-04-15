package releasecontroller

import (
	"context"
	"fmt"
)

// Tag represents a release tag with its pull specification
type Tag struct {
	Name        string `json:"name"`
	Phase       string `json:"phase"`
	PullSpec    string `json:"pullSpec"`
	DownloadURL string `json:"downloadURL"`
}

// TagsResponse is the response from the /api/v1/releasestream/{stream}/tags endpoint
type TagsResponse struct {
	Name string `json:"name"`
	Tags []Tag  `json:"tags"`
}

// GetTagsForStream fetches all tags for a specific release stream
// streamName is the release stream (e.g., "4-stable")
func (c *client) GetTagsForStream(ctx context.Context, streamName string) ([]Tag, error) {
	path := fmt.Sprintf("/api/v1/releasestream/%s/tags", streamName)

	var response TagsResponse
	if err := c.GetJSON(ctx, path, &response); err != nil {
		return nil, fmt.Errorf("failed to get tags for stream %s: %w", streamName, err)
	}

	return response.Tags, nil
}

// GetPullSpec finds the pull specification for a specific release name
// This searches across all streams in this endpoint to find the release
func (c *client) GetPullSpec(ctx context.Context, releaseName string) (string, error) {
	// First find which stream contains this release
	locations, err := c.FindRelease(ctx, releaseName)
	if err != nil {
		return "", fmt.Errorf("failed to find release %s: %w", releaseName, err)
	}

	if len(locations) == 0 {
		return "", fmt.Errorf("release %s not found", releaseName)
	}

	// Use the first location (there should typically only be one per endpoint)
	location := locations[0]

	// Get tags for that stream
	tags, err := c.GetTagsForStream(ctx, location.StreamName)
	if err != nil {
		return "", fmt.Errorf("failed to get tags for stream %s: %w", location.StreamName, err)
	}

	// Find the matching tag
	for _, tag := range tags {
		if tag.Name == releaseName {
			return tag.PullSpec, nil
		}
	}

	return "", fmt.Errorf("release %s found in stream %s but no matching tag", releaseName, location.StreamName)
}

// GetPullSpecForLocation gets the pull specification for a release at a specific location
func (c *client) GetPullSpecForLocation(ctx context.Context, location ReleaseLocation, releaseName string) (string, error) {
	tags, err := c.GetTagsForStream(ctx, location.StreamName)
	if err != nil {
		return "", fmt.Errorf("failed to get tags for stream %s: %w", location.StreamName, err)
	}

	for _, tag := range tags {
		if tag.Name == releaseName {
			return tag.PullSpec, nil
		}
	}

	return "", fmt.Errorf("release %s not found in stream %s tags", releaseName, location.StreamName)
}

// GetTagForRelease gets the full Tag information for a specific release
func (c *client) GetTagForRelease(ctx context.Context, releaseName string) (*Tag, error) {
	// First find which stream contains this release
	locations, err := c.FindRelease(ctx, releaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to find release %s: %w", releaseName, err)
	}

	if len(locations) == 0 {
		return nil, fmt.Errorf("release %s not found", releaseName)
	}

	// Use the first location
	location := locations[0]

	// Get tags for that stream
	tags, err := c.GetTagsForStream(ctx, location.StreamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags for stream %s: %w", location.StreamName, err)
	}

	// Find the matching tag
	for _, tag := range tags {
		if tag.Name == releaseName {
			return &tag, nil
		}
	}

	return nil, fmt.Errorf("release %s found in stream %s but no matching tag", releaseName, location.StreamName)
}
