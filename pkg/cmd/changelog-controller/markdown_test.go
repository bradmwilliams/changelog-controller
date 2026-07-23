package changelog_controller

import (
	"strings"
	"testing"
	"time"
)

func TestFormatChangeLogMarkdown(t *testing.T) {
	cl := &ChangeLog{
		From: ChangeLogReleaseInfo{
			Name:    "4.18.2",
			Created: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Digest:  "sha256:abc",
		},
		To: ChangeLogReleaseInfo{
			Name:    "4.18.3",
			Created: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			Digest:  "sha256:def",
		},
		Components: []ChangeLogComponentInfo{
			{Name: "Kubernetes", Version: "v1.31.1", From: "v1.31.0", DiffURL: "https://github.com/kubernetes/kubernetes/compare/v1.31.0...v1.31.1"},
		},
		UpdatedImages: []ChangeLogImageInfo{
			{
				Name: "cluster-etcd-operator",
				Path: "https://github.com/openshift/cluster-etcd-operator",
				Commits: []CommitInfo{
					{Subject: "Fix backup rotation", PullURL: "https://github.com/openshift/cluster-etcd-operator/pull/123"},
				},
			},
		},
	}

	md := FormatChangeLogMarkdown(cl)

	checks := []string{
		"# 4.18.2 → 4.18.3",
		"## Components",
		"**Kubernetes**: v1.31.0 → v1.31.1",
		"[diff](",
		"## Updated Images",
		"### [cluster-etcd-operator](",
		"- [Fix backup rotation](",
	}
	for _, want := range checks {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q\ngot:\n%s", want, md)
		}
	}
}

func TestFormatChangeLogMarkdown_Empty(t *testing.T) {
	cl := &ChangeLog{
		From: ChangeLogReleaseInfo{Name: "4.18.2"},
		To:   ChangeLogReleaseInfo{Name: "4.18.3"},
	}

	md := FormatChangeLogMarkdown(cl)
	if !strings.Contains(md, "# 4.18.2 → 4.18.3") {
		t.Errorf("expected header, got:\n%s", md)
	}
	if strings.Contains(md, "## Components") {
		t.Error("should not include Components section when empty")
	}
}

func TestFormatRpmDiffMarkdown(t *testing.T) {
	diff := &RpmDiff{
		Changed: map[string]RpmChangedDiff{
			"kernel": {Old: "5.14.0-427", New: "5.14.0-428"},
		},
		Added: map[string]string{
			"new-pkg": "1.0.0",
		},
		Removed: map[string]string{
			"old-pkg": "2.0.0",
		},
	}

	md := FormatRpmDiffMarkdown(diff)

	checks := []string{
		"## RPM Package Changes",
		"### Changed",
		"**kernel**: 5.14.0-427 → 5.14.0-428",
		"### Added",
		"**new-pkg**: 1.0.0",
		"### Removed",
		"**old-pkg**: 2.0.0",
	}
	for _, want := range checks {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q\ngot:\n%s", want, md)
		}
	}
}

func TestFormatRpmDiffMarkdown_Nil(t *testing.T) {
	if got := FormatRpmDiffMarkdown(nil); got != "" {
		t.Errorf("expected empty string for nil diff, got: %q", got)
	}
}
