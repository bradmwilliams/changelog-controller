package main

import "time"

// Release controller API types (local copies to avoid importing pkg/release-controller)

type APIRelease struct {
	Name string   `json:"name"`
	Tags []APITag `json:"tags"`
}

type APITag struct {
	Name        string `json:"name"`
	Phase       string `json:"phase"`
	PullSpec    string `json:"pullSpec"`
	DownloadURL string `json:"downloadURL"`
}

type APIReleaseInfo struct {
	Name             string                   `json:"name"`
	Phase            string                   `json:"phase"`
	Results          *VerificationJobsSummary `json:"results,omitempty"`
	UpgradesTo       []UpgradeHistory         `json:"upgradesTo,omitempty"`
	UpgradesFrom     []UpgradeHistory         `json:"upgradesFrom,omitempty"`
	ChangeLog        []byte                   `json:"changeLog,omitempty"`
	ChangeLogJson    *ChangeLog               `json:"changeLogJson,omitempty"`
	NodeImageStreams []APINodeImageStream     `json:"nodeImageStreams,omitempty"`
}

type APINodeImageStream struct {
	Name    string   `json:"name"`
	Tag     string   `json:"tag"`
	RpmDiff *RpmDiff `json:"rpmDiff,omitempty"`
}

type VerificationJobsSummary struct {
	BlockingJobs  VerificationStatusMap `json:"blockingJobs,omitempty"`
	InformingJobs VerificationStatusMap `json:"informingJobs,omitempty"`
}

type VerificationStatusMap map[string]*VerificationStatus

type VerificationStatus struct {
	State               string   `json:"state"`
	URL                 string   `json:"url"`
	Retries             int      `json:"retries,omitempty"`
	PreviousAttemptURLs []string `json:"previousAttemptURLs,omitempty"`
}

type UpgradeHistory struct {
	From    string                   `json:"from"`
	To      string                   `json:"to"`
	Success int                      `json:"success"`
	Failure int                      `json:"failure"`
	Total   int                      `json:"total"`
	History map[string]UpgradeResult `json:"history,omitempty"`
}

type UpgradeResult struct {
	State string `json:"state"`
	URL   string `json:"url"`
}

// Changelog controller API types

type ChangelogResponse struct {
	From          string     `json:"from"`
	To            string     `json:"to"`
	FromPullSpec  string     `json:"fromPullSpec,omitempty"`
	ToPullSpec    string     `json:"toPullSpec,omitempty"`
	Changelog     *ChangeLog `json:"changelog"`
	RpmDiff       *RpmDiff   `json:"rpmDiff,omitempty"`
	RpmDiffStatus string     `json:"rpmDiffStatus,omitempty"`
}

type ChangeLog struct {
	From ReleaseInfo `json:"from"`
	To   ReleaseInfo `json:"to"`

	Components    []ComponentInfo `json:"components,omitempty"`
	NewImages     []ImageInfo     `json:"newImages,omitempty"`
	RemovedImages []ImageInfo     `json:"removedImages,omitempty"`
	RebuiltImages []ImageInfo     `json:"rebuiltImages,omitempty"`
	UpdatedImages []ImageInfo     `json:"updatedImages,omitempty"`
}

type ReleaseInfo struct {
	Name         string    `json:"name"`
	Created      time.Time `json:"created"`
	Digest       string    `json:"digest"`
	PromotedFrom string    `json:"promotedFrom,omitempty"`
}

type ComponentInfo struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	VersionURL string `json:"versionUrl,omitempty"`
	From       string `json:"from,omitempty"`
	FromURL    string `json:"fromUrl,omitempty"`
	DiffURL    string `json:"diffUrl,omitempty"`
}

type ImageInfo struct {
	Name          string       `json:"name"`
	Path          string       `json:"path"`
	ShortCommit   string       `json:"shortCommit,omitempty"`
	Commit        string       `json:"commit,omitempty"`
	ImageRef      string       `json:"imageRef,omitempty"`
	Commits       []CommitInfo `json:"commits,omitempty"`
	FullChangeLog string       `json:"fullChangeLog,omitempty"`
}

type CommitInfo struct {
	Bugs      map[string]string `json:"bugs,omitempty"`
	Issues    map[string]string `json:"issues,omitempty"`
	Subject   string            `json:"subject,omitempty"`
	PullID    int               `json:"pullID,omitempty"`
	PullURL   string            `json:"pullURL,omitempty"`
	CommitID  string            `json:"commitID,omitempty"`
	CommitURL string            `json:"commitURL,omitempty"`
}

type RpmDiff struct {
	Changed map[string]RpmChangedDiff `json:"changed,omitempty"`
	Added   map[string]string         `json:"added,omitempty"`
	Removed map[string]string         `json:"removed,omitempty"`
}

type RpmChangedDiff struct {
	Old string `json:"old,omitempty"`
	New string `json:"new,omitempty"`
}

// Qualifiers API response
type QualifiersResponse struct {
	Qualifiers map[string]Qualifier `json:"qualifiers,omitempty"`
}

type Qualifier struct {
	AggregateState string `json:"aggregateState"`
	BadgeName      string `json:"badgeName"`
	BadgeEarned    bool   `json:"badgeEarned"`
}

// AllStreams API response
type AllStreamsResponse map[string][]StreamTag

type StreamTag struct {
	Name  string `json:"name"`
	Phase string `json:"phase"`
}
