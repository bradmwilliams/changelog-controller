package changelog_controller

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bradmwilliams/changelog-controller/pkg/storage"
	"github.com/bradmwilliams/changelog-controller/pkg/storage/filesystem"
)

func TestReleaseExtractor_GetSetCache(t *testing.T) {
	extractor := NewReleaseExtractor(nil)
	ctx := context.Background()

	pullSpec := "registry.ci.openshift.org/ocp/release@sha256:test"

	// Get on empty cache
	snap, found := extractor.Get(ctx, pullSpec)
	if found {
		t.Error("Get() found = true, want false for empty cache")
	}
	if snap != nil {
		t.Error("Get() returned non-nil snapshot")
	}

	// Manually populate cache to test Get without running oc
	extractor.cache[pullSpec] = &storage.ReleaseSnapshot{
		PullSpec:    pullSpec,
		Manifest:    json.RawMessage(`{"test": true}`),
		RPMPackages: map[string]string{"kernel": "5.14.0-427.el9"},
		ExtractedAt: time.Now(),
	}

	snap, found = extractor.Get(ctx, pullSpec)
	if !found {
		t.Fatal("Get() found = false after cache population")
	}
	if snap.PullSpec != pullSpec {
		t.Errorf("Get() pullSpec = %q, want %q", snap.PullSpec, pullSpec)
	}
}

func TestReleaseExtractor_GetFromStorage(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Join(os.TempDir(), "test-extractor-storage")
	defer os.RemoveAll(tempDir)

	ss, err := filesystem.NewSnapshotStorage(ctx, tempDir)
	if err != nil {
		t.Fatalf("NewSnapshotStorage() error = %v", err)
	}
	defer ss.Close()

	pullSpec := "registry.ci.openshift.org/ocp/release@sha256:stored"

	// Write directly to storage (bypassing extractor)
	snapshot := &storage.ReleaseSnapshot{
		PullSpec:    pullSpec,
		Manifest:    json.RawMessage(`{"stored": true}`),
		RPMPackages: map[string]string{"cri-o": "1.30.0-1.el9"},
		ExtractedAt: time.Now(),
	}
	if err := ss.Set(ctx, pullSpec, snapshot); err != nil {
		t.Fatalf("storage Set() error = %v", err)
	}

	// Create extractor backed by this storage
	extractor := NewReleaseExtractor(ss)

	// Get should find it in storage and populate the in-memory cache
	snap, found := extractor.Get(ctx, pullSpec)
	if !found {
		t.Fatal("Get() found = false, want true from storage backend")
	}
	if snap.PullSpec != pullSpec {
		t.Errorf("Get() pullSpec = %q, want %q", snap.PullSpec, pullSpec)
	}

	// Verify it was promoted to in-memory cache
	extractor.mu.RLock()
	_, inMemory := extractor.cache[pullSpec]
	extractor.mu.RUnlock()
	if !inMemory {
		t.Error("snapshot not promoted to in-memory cache after storage hit")
	}
}

func TestReleaseExtractor_WarmCache(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Join(os.TempDir(), "test-extractor-warm")
	defer os.RemoveAll(tempDir)

	ss, err := filesystem.NewSnapshotStorage(ctx, tempDir)
	if err != nil {
		t.Fatalf("NewSnapshotStorage() error = %v", err)
	}
	defer ss.Close()

	specs := []string{
		"registry.ci.openshift.org/ocp/release@sha256:warm1",
		"registry.ci.openshift.org/ocp/release@sha256:warm2",
	}
	for _, spec := range specs {
		snap := &storage.ReleaseSnapshot{
			PullSpec:    spec,
			Manifest:    json.RawMessage(`{}`),
			ExtractedAt: time.Now(),
		}
		if err := ss.Set(ctx, spec, snap); err != nil {
			t.Fatalf("storage Set() error = %v", err)
		}
	}

	extractor := NewReleaseExtractor(ss)

	if err := extractor.WarmCache(ctx); err != nil {
		t.Fatalf("WarmCache() error = %v", err)
	}

	extractor.mu.RLock()
	cacheLen := len(extractor.cache)
	extractor.mu.RUnlock()

	if cacheLen != len(specs) {
		t.Errorf("cache has %d entries after warm, want %d", cacheLen, len(specs))
	}

	for _, spec := range specs {
		snap, found := extractor.Get(ctx, spec)
		if !found {
			t.Errorf("Get(%s) found = false after warm", spec)
		}
		if snap.PullSpec != spec {
			t.Errorf("Get(%s) pullSpec = %q", spec, snap.PullSpec)
		}
	}
}

func TestReleaseExtractor_WarmCacheNilStorage(t *testing.T) {
	extractor := NewReleaseExtractor(nil)
	if err := extractor.WarmCache(context.Background()); err != nil {
		t.Errorf("WarmCache() with nil storage should not error, got %v", err)
	}
}
