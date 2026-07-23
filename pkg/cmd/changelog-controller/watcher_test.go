package changelog_controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	rc "github.com/bradmwilliams/changelog-controller/pkg/release-controller"
	"github.com/openshift/library-go/pkg/operator/events"
	"k8s.io/utils/clock"
)

func TestReleaseWatcher_DetectsNewReleases(t *testing.T) {
	var mu sync.Mutex
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count := callCount
		mu.Unlock()

		switch r.URL.Path {
		case "/api/v1/releasestreams/ready":
			var resp map[string][]string
			if count == 0 {
				resp = map[string][]string{
					"4-stable": {"4.18.2", "4.18.1", "4.18.0"},
				}
			} else {
				// Phase 2: new release appears
				resp = map[string][]string{
					"4-stable": {"4.18.3", "4.18.2", "4.18.1", "4.18.0"},
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		case "/api/v1/releasestream/4-stable/latest":
			// Latest Accepted release is the predecessor of the new Ready release
			resp := rc.Tag{
				Name:     "4.18.2",
				Phase:    "Accepted",
				PullSpec: "registry.ci.openshift.org/ocp/release:4.18.2",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		case "/api/v1/releasestreams/all":
			resp := map[string][]string{
				"4-stable": {"4.18.3", "4.18.2", "4.18.1", "4.18.0"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		case "/api/v1/releasestream/4-stable/tags":
			resp := rc.TagsResponse{
				Name: "4-stable",
				Tags: []rc.Tag{
					{Name: "4.18.3", PullSpec: "registry.ci.openshift.org/ocp/release:4.18.3"},
					{Name: "4.18.2", PullSpec: "registry.ci.openshift.org/ocp/release:4.18.2"},
					{Name: "4.18.1", PullSpec: "registry.ci.openshift.org/ocp/release:4.18.1"},
					{Name: "4.18.0", PullSpec: "registry.ci.openshift.org/ocp/release:4.18.0"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create a client pointing at the test server
	endpoint := &rc.Endpoint{
		Architecture: rc.ArchitectureAMD64,
		Product:      rc.ProductOCP,
		URL:          server.URL,
	}
	client, err := rc.NewClient(endpoint, &rc.ClientConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	mcClient := &rc.MultiClient{}
	mcClient.AddClientForTesting(rc.ProductOCP, rc.ArchitectureAMD64, client)

	// Create controller that captures enqueue calls
	recorder := events.NewInMemoryRecorder("test", clock.RealClock{})
	cache := NewChangelogCache()
	controller := NewReleasePayloadChangelogController(recorder, cache)

	// Create watcher with short poll interval for testing
	watcher := NewReleaseWatcher(mcClient, controller, nil, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Seed initial state
	watcher.seedInitialState(ctx)

	// Verify nothing was enqueued during seeding
	queueStats := controller.QueueStats()
	if queueStats.Depth != 0 {
		t.Errorf("expected 0 items in queue after seeding, got %d", queueStats.Depth)
	}

	// Simulate phase 2: new release appears
	mu.Lock()
	callCount = 1
	mu.Unlock()

	// Poll once
	watcher.poll(ctx)

	// Verify the enqueued item is the right pair (4.18.2 -> 4.18.3)
	queueStats = controller.QueueStats()
	if len(queueStats.Items) != 1 {
		t.Fatalf("expected 1 enqueued request, got %d", len(queueStats.Items))
	}
	req := queueStats.Items[0].Request
	if req.From != "registry.ci.openshift.org/ocp/release:4.18.2" {
		t.Errorf("expected from=registry.ci.openshift.org/ocp/release:4.18.2, got %s", req.From)
	}
	if req.To != "registry.ci.openshift.org/ocp/release:4.18.3" {
		t.Errorf("expected to=registry.ci.openshift.org/ocp/release:4.18.3, got %s", req.To)
	}
}

func TestReleaseWatcher_NoDuplicateEnqueue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/releasestreams/ready":
			resp := map[string][]string{
				"4-stable": {"4.18.2", "4.18.1", "4.18.0"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		case "/api/v1/releasestreams/all":
			resp := map[string][]string{
				"4-stable": {"4.18.2", "4.18.1", "4.18.0"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	endpoint := &rc.Endpoint{
		Architecture: rc.ArchitectureAMD64,
		Product:      rc.ProductOCP,
		URL:          server.URL,
	}
	client, err := rc.NewClient(endpoint, &rc.ClientConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	mcClient := &rc.MultiClient{}
	mcClient.AddClientForTesting(rc.ProductOCP, rc.ArchitectureAMD64, client)

	recorder := events.NewInMemoryRecorder("test", clock.RealClock{})
	cache := NewChangelogCache()
	controller := NewReleasePayloadChangelogController(recorder, cache)
	watcher := NewReleaseWatcher(mcClient, controller, nil, 100*time.Millisecond)

	ctx := context.Background()

	// Seed and poll multiple times — nothing new should be enqueued
	watcher.seedInitialState(ctx)
	watcher.poll(ctx)
	watcher.poll(ctx)
	watcher.poll(ctx)

	queueStats := controller.QueueStats()
	if queueStats.Depth != 0 {
		t.Errorf("expected 0 items in queue (no new releases), got %d", queueStats.Depth)
	}
}
