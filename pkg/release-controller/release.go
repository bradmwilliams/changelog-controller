package releasecontroller

import (
	"context"
	"fmt"
)

// JobState represents the state of a CI job
type JobState string

const (
	JobStateSucceeded JobState = "Succeeded"
	JobStateFailed    JobState = "Failed"
	JobStatePending   JobState = "Pending"
	JobStateRunning   JobState = "Running"
)

// JobResult contains information about a CI job result
type JobResult struct {
	State               JobState `json:"state"`
	URL                 string   `json:"url"`
	Retries             int      `json:"retries,omitempty"`
	PreviousAttemptURLs []string `json:"previousAttemptURLs,omitempty"`
}

// JobResults contains all job results organized by blocking and informing
type JobResults struct {
	BlockingJobs  map[string]JobResult `json:"blockingJobs"`
	InformingJobs map[string]JobResult `json:"informingJobs"`
}

// UpgradeHistory represents upgrade test history from one version to another
type UpgradeHistory struct {
	From    string               `json:"From"`
	To      string               `json:"To"`
	Success int                  `json:"Success"`
	Failure int                  `json:"Failure"`
	Total   int                  `json:"Total"`
	History map[string]JobResult `json:"History,omitempty"`
}

// ReleaseRef represents a reference to a release with digest
type ReleaseRef struct {
	Name    string `json:"name"`
	Created string `json:"created,omitempty"`
	Digest  string `json:"digest,omitempty"`
}

// ChangeLogComponent represents a component in the changelog
type ChangeLogComponent struct {
	Name       string `json:"name"`
	Version    string `json:"version,omitempty"`
	From       string `json:"from,omitempty"`
	VersionUrl string `json:"versionUrl,omitempty"`
	FromUrl    string `json:"fromUrl,omitempty"`
	DiffUrl    string `json:"diffUrl,omitempty"`
}

// Commit represents a git commit in an updated image
type Commit struct {
	Issues  map[string]string `json:"issues,omitempty"`
	Subject string            `json:"subject"`
	PullID  int               `json:"pullID,omitempty"`
	PullURL string            `json:"pullURL,omitempty"`
}

// UpdatedImage represents an image that was updated with new commits
type UpdatedImage struct {
	Name          string   `json:"name"`
	Path          string   `json:"path"`
	Commits       []Commit `json:"commits,omitempty"`
	FullChangeLog string   `json:"fullChangeLog,omitempty"`
}

// RebuiltImage represents an image that was rebuilt
type RebuiltImage struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ShortCommit string `json:"shortCommit,omitempty"`
	Commit      string `json:"commit,omitempty"`
	ImageRef    string `json:"imageRef,omitempty"`
}

// ChangeLogJson contains structured changelog information
type ChangeLogJson struct {
	From          *ReleaseRef          `json:"from"`
	To            *ReleaseRef          `json:"to"`
	Components    []ChangeLogComponent `json:"components,omitempty"`
	UpdatedImages []UpdatedImage       `json:"updatedImages,omitempty"`
	RebuiltImages []RebuiltImage       `json:"rebuiltImages,omitempty"`
}

// ReleaseInfo contains detailed information about a specific release
type ReleaseInfo struct {
	Name          string           `json:"name"`
	Phase         string           `json:"phase"`
	ChangeLog     string           `json:"changeLog,omitempty"`     // Markdown/text changelog
	ChangeLogJson *ChangeLogJson   `json:"changeLogJson,omitempty"` // Structured changelog
	Results       *JobResults      `json:"results,omitempty"`
	UpgradesTo    []UpgradeHistory `json:"upgradesTo,omitempty"`
	UpgradesFrom  []UpgradeHistory `json:"upgradesFrom,omitempty"`
}

// GetReleaseInfo fetches detailed information for a specific release
// streamName is the release stream (e.g., "4-stable")
// releaseName is the specific release tag (e.g., "4.21.10")
func (c *client) GetReleaseInfo(ctx context.Context, streamName, releaseName string) (*ReleaseInfo, error) {
	path := fmt.Sprintf("/api/v1/releasestream/%s/release/%s", streamName, releaseName)

	var info ReleaseInfo
	if err := c.GetJSON(ctx, path, &info); err != nil {
		return nil, fmt.Errorf("failed to get release info for %s in stream %s: %w", releaseName, streamName, err)
	}

	return &info, nil
}

// GetReleaseInfoForLocation is a convenience method that fetches release info
// using a ReleaseLocation obtained from FindRelease
func (c *client) GetReleaseInfoForLocation(ctx context.Context, location ReleaseLocation, releaseName string) (*ReleaseInfo, error) {
	return c.GetReleaseInfo(ctx, location.StreamName, releaseName)
}
