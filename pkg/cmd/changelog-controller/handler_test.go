package changelog_controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	rc "github.com/bradmwilliams/changelog-controller/pkg/release-controller"
)

func TestHandleChangelog_Validation(t *testing.T) {
	// Create mock release-controller server
	rcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch r.URL.Path {
		case "/api/v1/releasestreams/all":
			w.Write([]byte(`{
				"4-stable": ["2.0.0", "1.0.0"]
			}`))
		case "/api/v1/releasestream/4-stable/tags":
			w.Write([]byte(`{
				"name": "4-stable",
				"tags": [
					{
						"name": "1.0.0",
						"phase": "Accepted",
						"pullSpec": "quay.io/test/release:1.0.0-x86_64",
						"downloadURL": "https://example.com/1.0.0"
					},
					{
						"name": "2.0.0",
						"phase": "Accepted",
						"pullSpec": "quay.io/test/release:2.0.0-x86_64",
						"downloadURL": "https://example.com/2.0.0"
					}
				]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer rcServer.Close()

	// Create release-controller client
	endpoint := &rc.Endpoint{
		Architecture: rc.ArchitectureAMD64,
		Product:      rc.ProductOCP,
		URL:          rcServer.URL,
	}

	client, err := rc.NewClient(endpoint, rc.DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create release-controller client: %v", err)
	}
	defer client.Close()

	// Create a MultiClient for testing with just this one client
	multiClient := &rc.MultiClient{}
	multiClient.AddClientForTesting(rc.ProductOCP, rc.ArchitectureAMD64, client)

	ctx := context.Background()
	cache := NewChangelogCache()

	// Add a test changelog to the cache using PullSpecs as keys
	cache.Set(ctx, "quay.io/test/release:1.0.0-x86_64", "quay.io/test/release:2.0.0-x86_64", "# Changelog\n\nTest changes")

	opts := &Options{
		cache:    cache,
		rcClient: multiClient,
	}

	tests := []struct {
		name                 string
		queryParams          string
		expectedStatus       int
		expectedContentType  string
		expectedBodyContains string
	}{
		{
			name:                 "default format is json",
			queryParams:          "from=1.0.0&to=2.0.0&arch=amd64",
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"from":"1.0.0"`,
		},
		{
			name:                 "explicit json format",
			queryParams:          "from=1.0.0&to=2.0.0&arch=amd64&format=json",
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"changelog":`,
		},
		{
			name:                 "markdown format",
			queryParams:          "from=1.0.0&to=2.0.0&arch=amd64&format=md",
			expectedStatus:       http.StatusOK,
			expectedContentType:  "text/markdown",
			expectedBodyContains: "# Changelog",
		},
		{
			name:                 "invalid format",
			queryParams:          "from=1.0.0&to=2.0.0&arch=amd64&format=xml",
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "application/json",
			expectedBodyContains: `"error":`,
		},
		{
			name:                 "missing from parameter",
			queryParams:          "to=2.0.0&arch=amd64",
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "application/json",
			expectedBodyContains: `"error":`,
		},
		{
			name:                 "missing to parameter",
			queryParams:          "from=1.0.0&arch=amd64",
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "application/json",
			expectedBodyContains: `"error":`,
		},
		{
			name:                 "missing arch parameter",
			queryParams:          "from=1.0.0&to=2.0.0",
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "application/json",
			expectedBodyContains: `"error":`,
		},
		{
			name:                 "invalid arch parameter",
			queryParams:          "from=1.0.0&to=2.0.0&arch=invalid",
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "application/json",
			expectedBodyContains: `"error":`,
		},
		{
			name:                 "invalid from semver",
			queryParams:          "from=invalid&to=2.0.0&arch=amd64",
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "application/json",
			expectedBodyContains: `"error":`,
		},
		{
			name:                 "invalid to semver",
			queryParams:          "from=1.0.0&to=invalid&arch=amd64",
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "application/json",
			expectedBodyContains: `"error":`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/changelog?"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			opts.handleChangelog(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			contentType := resp.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, tt.expectedContentType) {
				t.Errorf("Expected Content-Type %q, got %q", tt.expectedContentType, contentType)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.expectedBodyContains) {
				t.Errorf("Expected body to contain %q, got: %s", tt.expectedBodyContains, body)
			}
		})
	}
}

func TestHandleChangelog_JSONEscaping(t *testing.T) {
	// Create mock release-controller server
	rcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch r.URL.Path {
		case "/api/v1/releasestreams/all":
			w.Write([]byte(`{
				"4-stable": ["2.0.0", "1.0.0"]
			}`))
		case "/api/v1/releasestream/4-stable/tags":
			w.Write([]byte(`{
				"name": "4-stable",
				"tags": [
					{
						"name": "1.0.0",
						"phase": "Accepted",
						"pullSpec": "quay.io/test/release:1.0.0-x86_64",
						"downloadURL": "https://example.com/1.0.0"
					},
					{
						"name": "2.0.0",
						"phase": "Accepted",
						"pullSpec": "quay.io/test/release:2.0.0-x86_64",
						"downloadURL": "https://example.com/2.0.0"
					}
				]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer rcServer.Close()

	// Create release-controller client
	endpoint := &rc.Endpoint{
		Architecture: rc.ArchitectureAMD64,
		Product:      rc.ProductOCP,
		URL:          rcServer.URL,
	}

	client, err := rc.NewClient(endpoint, rc.DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create release-controller client: %v", err)
	}
	defer client.Close()

	// Create a MultiClient for testing with just this one client
	multiClient := &rc.MultiClient{}
	multiClient.AddClientForTesting(rc.ProductOCP, rc.ArchitectureAMD64, client)

	ctx := context.Background()
	cache := NewChangelogCache()

	// Add a changelog with special characters that need escaping
	changelog := `Line 1
Line 2 with "quotes"
Line 3 with \backslash
Line 4 with	tab`

	cache.Set(ctx, "quay.io/test/release:1.0.0-x86_64", "quay.io/test/release:2.0.0-x86_64", changelog)

	opts := &Options{
		cache:    cache,
		rcClient: multiClient,
	}

	req := httptest.NewRequest("GET", "/changelog?from=1.0.0&to=2.0.0&arch=amd64", nil)
	w := httptest.NewRecorder()

	opts.handleChangelog(w, req)

	body := w.Body.String()

	// Verify escaping
	if !strings.Contains(body, `\n`) {
		t.Error("Expected newlines to be escaped as \\n")
	}
	if !strings.Contains(body, `\"`) {
		t.Error("Expected quotes to be escaped as \\\"")
	}
	if !strings.Contains(body, `\\`) {
		t.Error("Expected backslashes to be escaped as \\\\")
	}
	if !strings.Contains(body, `\t`) {
		t.Error("Expected tabs to be escaped as \\t")
	}
}

func TestWriteChangelogResponse(t *testing.T) {
	opts := &Options{}

	tests := []struct {
		name            string
		from            string
		to              string
		changelog       string
		format          string
		fromPullSpec    string
		toPullSpec      string
		expectedCT      string
		expectedContain string
	}{
		{
			name:            "json without pullspecs",
			from:            "1.0.0",
			to:              "2.0.0",
			changelog:       "Simple changelog",
			format:          "json",
			fromPullSpec:    "",
			toPullSpec:      "",
			expectedCT:      "application/json",
			expectedContain: `"changelog":"Simple changelog"`,
		},
		{
			name:            "json with pullspecs",
			from:            "4.21.9",
			to:              "4.21.10",
			changelog:       "Bugfixes",
			format:          "json",
			fromPullSpec:    "quay.io/openshift-release-dev/ocp-release:4.21.9-x86_64",
			toPullSpec:      "quay.io/openshift-release-dev/ocp-release:4.21.10-x86_64",
			expectedCT:      "application/json",
			expectedContain: `"fromPullSpec":"quay.io/openshift-release-dev/ocp-release:4.21.9-x86_64"`,
		},
		{
			name:            "markdown format",
			from:            "1.0.0",
			to:              "2.0.0",
			changelog:       "Raw changelog text",
			format:          "md",
			fromPullSpec:    "",
			toPullSpec:      "",
			expectedCT:      "text/markdown",
			expectedContain: "Raw changelog text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			opts.writeChangelogResponse(w, tt.from, tt.to, tt.changelog, tt.format, "amd64", tt.fromPullSpec, tt.toPullSpec, nil)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, tt.expectedCT) {
				t.Errorf("Expected Content-Type %q, got %q", tt.expectedCT, contentType)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.expectedContain) {
				t.Errorf("Expected body to contain %q, got: %s", tt.expectedContain, body)
			}
		})
	}
}

func TestWriteChangelogResponse_StructuredJSON(t *testing.T) {
	opts := &Options{}

	changelogJSON := `{"from":{"name":"4.18.2","created":"2026-01-01T00:00:00Z","digest":"sha256:abc"},"to":{"name":"4.18.3","created":"2026-01-02T00:00:00Z","digest":"sha256:def"},"components":[{"name":"Kubernetes","version":"v1.31.0","from":"v1.30.0"}],"updatedImages":[{"name":"cluster-etcd-operator","path":"https://github.com/openshift/cluster-etcd-operator","commits":[{"subject":"Fix backup","pullID":123}]}]}`

	w := httptest.NewRecorder()
	opts.writeChangelogResponse(w, "4.18.2", "4.18.3", changelogJSON, "json", "amd64", "registry/from", "registry/to", nil)

	resp := w.Result()
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	changelog, ok := result["changelog"].(map[string]interface{})
	if !ok {
		t.Fatal("changelog field is not a JSON object")
	}

	// Verify the changelog is structured (has typed from/to objects, not raw string)
	fromObj, ok := changelog["from"].(map[string]interface{})
	if !ok {
		t.Fatal("changelog.from is not a JSON object")
	}
	if fromObj["name"] != "4.18.2" {
		t.Errorf("changelog.from.name = %v, want 4.18.2", fromObj["name"])
	}

	toObj, ok := changelog["to"].(map[string]interface{})
	if !ok {
		t.Fatal("changelog.to is not a JSON object")
	}
	if toObj["name"] != "4.18.3" {
		t.Errorf("changelog.to.name = %v, want 4.18.3", toObj["name"])
	}

	components, ok := changelog["components"].([]interface{})
	if !ok {
		t.Fatal("changelog.components is not an array")
	}
	if len(components) != 1 {
		t.Errorf("changelog.components has %d entries, want 1", len(components))
	}

	updatedImages, ok := changelog["updatedImages"].([]interface{})
	if !ok {
		t.Fatal("changelog.updatedImages is not an array")
	}
	if len(updatedImages) != 1 {
		t.Errorf("changelog.updatedImages has %d entries, want 1", len(updatedImages))
	}
}

func TestWriteChangelogResponse_CoreOSEnrichment(t *testing.T) {
	opts := &Options{}

	changelogJSON := `{"from":{"name":"4.18.2","created":"2026-01-01T00:00:00Z","digest":"sha256:abc"},"to":{"name":"4.18.3","created":"2026-01-02T00:00:00Z","digest":"sha256:def"},"components":[{"name":"Red Hat Enterprise Linux CoreOS","version":"418.94.202501010000-0","from":"418.94.202412010000-0"},{"name":"Kubernetes","version":"v1.31.0","from":"v1.30.0"}]}`

	w := httptest.NewRecorder()
	opts.writeChangelogResponse(w, "4.18.2", "4.18.3", changelogJSON, "json", "amd64", "registry/from", "registry/to", nil)

	body := w.Body.String()

	// RHCOS component should have enrichment URLs
	if !strings.Contains(body, "releases-rhcos") {
		t.Error("expected RHCOS release browser URL in response")
	}
	if !strings.Contains(body, "versionUrl") {
		t.Error("expected versionUrl field in RHCOS component")
	}
	if !strings.Contains(body, "diff.html") {
		t.Error("expected diff URL in RHCOS component")
	}

	// Kubernetes component should not have URLs
	var result map[string]interface{}
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	changelog := result["changelog"].(map[string]interface{})
	components := changelog["components"].([]interface{})
	k8s := components[1].(map[string]interface{})
	if _, ok := k8s["versionUrl"]; ok {
		t.Error("Kubernetes component should not have versionUrl")
	}
}

func TestWriteChangelogResponse_FallbackForNonChangelogJSON(t *testing.T) {
	opts := &Options{}

	// Valid JSON but not a ChangeLog (no from/to names)
	arbitraryJSON := `{"key":"value","nested":{"a":1}}`

	w := httptest.NewRecorder()
	opts.writeChangelogResponse(w, "1.0.0", "2.0.0", arbitraryJSON, "json", "amd64", "", "", nil)

	var result map[string]interface{}
	if err := json.NewDecoder(w.Result().Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should preserve the raw JSON, not convert to an empty ChangeLog
	changelog, ok := result["changelog"].(map[string]interface{})
	if !ok {
		t.Fatal("changelog field is not a JSON object")
	}
	if changelog["key"] != "value" {
		t.Errorf("arbitrary JSON field not preserved: got %v", changelog["key"])
	}
}
