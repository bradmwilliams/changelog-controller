package changelog_controller

import "time"

// ChangeLog represents the structured output of
// oc adm release info --changelog --output=json
type ChangeLog struct {
	From ChangeLogReleaseInfo `json:"from"`
	To   ChangeLogReleaseInfo `json:"to"`

	Components    []ChangeLogComponentInfo `json:"components,omitempty"`
	NewImages     []ChangeLogImageInfo     `json:"newImages,omitempty"`
	RemovedImages []ChangeLogImageInfo     `json:"removedImages,omitempty"`
	RebuiltImages []ChangeLogImageInfo     `json:"rebuiltImages,omitempty"`
	UpdatedImages []ChangeLogImageInfo     `json:"updatedImages,omitempty"`
}

// ChangeLogReleaseInfo identifies a release in a changelog diff
type ChangeLogReleaseInfo struct {
	Name         string    `json:"name"`
	Created      time.Time `json:"created"`
	Digest       string    `json:"digest"`
	PromotedFrom string    `json:"promotedFrom,omitempty"`
}

// ChangeLogComponentInfo describes a version change for a platform component
type ChangeLogComponentInfo struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	VersionURL string `json:"versionUrl,omitempty"`
	From       string `json:"from,omitempty"`
	FromURL    string `json:"fromUrl,omitempty"`
	DiffURL    string `json:"diffUrl,omitempty"`
}

// ChangeLogImageInfo describes a container image that was added, removed,
// rebuilt, or updated between two releases
type ChangeLogImageInfo struct {
	Name          string       `json:"name"`
	Path          string       `json:"path"`
	ShortCommit   string       `json:"shortCommit,omitempty"`
	Commit        string       `json:"commit,omitempty"`
	ImageRef      string       `json:"imageRef,omitempty"`
	Commits       []CommitInfo `json:"commits,omitempty"`
	FullChangeLog string       `json:"fullChangeLog,omitempty"`
}

// CommitInfo represents a single git commit within an updated image
type CommitInfo struct {
	Bugs      map[string]string `json:"bugs,omitempty"`
	Issues    map[string]string `json:"issues,omitempty"`
	Subject   string            `json:"subject,omitempty"`
	PullID    int               `json:"pullID,omitempty"`
	PullURL   string            `json:"pullURL,omitempty"`
	CommitID  string            `json:"commitID,omitempty"`
	CommitURL string            `json:"commitURL,omitempty"`
}
