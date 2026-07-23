package changelog_controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/bradmwilliams/changelog-controller/pkg/storage"
	"k8s.io/klog/v2"
)

// ReleaseExtractor extracts and caches per-release metadata by running
// oc adm release info against a release pull spec to collect its
// component manifest and RPM package list
type ReleaseExtractor struct {
	mu      sync.RWMutex
	cache   map[string]*storage.ReleaseSnapshot
	storage storage.ReleaseSnapshotStorage
}

// NewReleaseExtractor creates an extractor backed by the given storage.
// If storage is nil, snapshots are kept in memory only
func NewReleaseExtractor(snapshotStorage storage.ReleaseSnapshotStorage) *ReleaseExtractor {
	return &ReleaseExtractor{
		cache:   make(map[string]*storage.ReleaseSnapshot),
		storage: snapshotStorage,
	}
}

// Get returns a cached snapshot for the given pull spec, checking
// in-memory cache first, then the storage backend
func (e *ReleaseExtractor) Get(ctx context.Context, pullSpec string) (*storage.ReleaseSnapshot, bool) {
	e.mu.RLock()
	snap, found := e.cache[pullSpec]
	e.mu.RUnlock()
	if found {
		return snap, true
	}

	if e.storage != nil {
		snap, found, err := e.storage.Get(ctx, pullSpec)
		if err != nil {
			klog.V(4).Infof("Extractor: storage lookup failed for %s: %v", pullSpec, err)
			return nil, false
		}
		if found {
			e.mu.Lock()
			e.cache[pullSpec] = snap
			e.mu.Unlock()
			return snap, true
		}
	}

	return nil, false
}

// Extract runs oc adm release info to collect the component manifest and
// RPM package list for the given pull spec. If the snapshot is already
// cached, this is a no-op
func (e *ReleaseExtractor) Extract(ctx context.Context, pullSpec string) error {
	if _, found := e.Get(ctx, pullSpec); found {
		klog.V(4).Infof("Extractor: snapshot already cached for %s", pullSpec)
		return nil
	}

	klog.V(4).Infof("Extractor: extracting release metadata for %s", pullSpec)

	oc, err := exec.LookPath("oc")
	if err != nil {
		return fmt.Errorf("failed to find oc: %w", err)
	}

	// Extract component manifest
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, oc, "adm", "release", "info", "--output=json", pullSpec)
	cmd.Stdout = out
	cmd.Stderr = errOut
	klog.V(4).Infof("Extractor: running %s", cmd.String())

	if err := cmd.Run(); err != nil {
		msg := errOut.String()
		if len(msg) == 0 {
			msg = err.Error()
		}
		return fmt.Errorf("oc adm release info failed for %s: %s", pullSpec, msg)
	}

	if !json.Valid(out.Bytes()) {
		return fmt.Errorf("oc returned invalid JSON for %s", pullSpec)
	}

	snapshot := &storage.ReleaseSnapshot{
		PullSpec:    pullSpec,
		Manifest:    json.RawMessage(out.Bytes()),
		ExtractedAt: time.Now(),
	}

	// Extract RPM package list (best-effort)
	rpmOut, rpmErrOut := &bytes.Buffer{}, &bytes.Buffer{}
	rpmCmd := exec.CommandContext(ctx, oc, "adm", "release", "info", "--rpmdb-cache=/tmp/rpmdb/", "--output=json", "--rpmdb", pullSpec)
	rpmCmd.Stdout = rpmOut
	rpmCmd.Stderr = rpmErrOut
	klog.V(4).Infof("Extractor: running %s", rpmCmd.String())

	if err := rpmCmd.Run(); err != nil {
		msg := rpmErrOut.String()
		if len(msg) == 0 {
			msg = err.Error()
		}
		klog.Warningf("Extractor: rpmdb extraction failed for %s: %s", pullSpec, msg)
	} else {
		var rpmList storage.RpmList
		if err := json.Unmarshal(rpmOut.Bytes(), &rpmList); err != nil {
			klog.Warningf("Extractor: failed to unmarshal rpmdb output for %s: %v", pullSpec, err)
		} else {
			snapshot.RPMList = &rpmList
		}
	}

	e.mu.Lock()
	e.cache[pullSpec] = snapshot
	e.mu.Unlock()

	if e.storage != nil {
		if err := e.storage.Set(ctx, pullSpec, snapshot); err != nil {
			klog.Warningf("Extractor: failed to persist snapshot for %s: %v", pullSpec, err)
		}
	}

	klog.Infof("Extractor: stored release snapshot for %s", pullSpec)
	return nil
}

// WarmCache loads all snapshots from the storage backend into the in-memory cache
func (e *ReleaseExtractor) WarmCache(ctx context.Context) error {
	if e.storage == nil {
		return nil
	}

	lister, ok := e.storage.(storage.ReleaseSnapshotLister)
	if !ok {
		klog.V(4).Info("Extractor: snapshot storage does not support listing, skipping warm")
		return nil
	}

	snapshots, err := lister.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	e.mu.Lock()
	for _, snap := range snapshots {
		e.cache[snap.PullSpec] = snap
	}
	e.mu.Unlock()

	klog.Infof("Extractor: warmed cache with %d snapshots", len(snapshots))
	return nil
}
