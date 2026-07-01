package changelog_controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	rc "github.com/bradmwilliams/changelog-controller/pkg/release-controller"
)

func TestHandleChangelog_FormatParameter(t *testing.T) {
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
			name:                 "missing from parameter with json format",
			queryParams:          "to=2.0.0&arch=amd64&format=json",
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "application/json",
			expectedBodyContains: `"error":`,
		},
		{
			name:                 "missing to parameter with md format",
			queryParams:          "from=1.0.0&arch=amd64&format=md",
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "text/plain",
			expectedBodyContains: "Error:",
		},
		{
			name:                 "missing arch parameter with json format",
			queryParams:          "from=1.0.0&to=2.0.0&format=json",
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "application/json",
			expectedBodyContains: `"error":`,
		},
		{
			name:                 "invalid arch parameter",
			queryParams:          "from=1.0.0&to=2.0.0&arch=invalid&format=json",
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "application/json",
			expectedBodyContains: `"error":`,
		},
		{
			name:                 "invalid semver with json format",
			queryParams:          "from=invalid&to=2.0.0&arch=amd64&format=json",
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "application/json",
			expectedBodyContains: `"error":`,
		},
		{
			name:                 "invalid semver with md format",
			queryParams:          "from=1.0.0&to=invalid&arch=amd64&format=md",
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "text/plain",
			expectedBodyContains: "Error:",
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
				t.Errorf("Expected Content-Type to start with %q, got %q", tt.expectedContentType, contentType)
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

	req := httptest.NewRequest("GET", "/changelog?from=1.0.0&to=2.0.0&arch=amd64&format=json", nil)
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
		name         string
		from         string
		to           string
		changelog    string
		format       string
		fromPullSpec string
		toPullSpec   string
		expectedCT   string
		expectedBody string
	}{
		{
			name:         "json format without pullspecs",
			from:         "1.0.0",
			to:           "2.0.0",
			changelog:    "Simple changelog",
			format:       "json",
			fromPullSpec: "",
			toPullSpec:   "",
			expectedCT:   "application/json",
			expectedBody: `{"from":"1.0.0","to":"2.0.0","changelog":"Simple changelog"}`,
		},
		{
			name:         "json format with pullspecs",
			from:         "4.21.9",
			to:           "4.21.10",
			changelog:    "Bugfixes",
			format:       "json",
			fromPullSpec: "quay.io/openshift-release-dev/ocp-release:4.21.9-x86_64",
			toPullSpec:   "quay.io/openshift-release-dev/ocp-release:4.21.10-x86_64",
			expectedCT:   "application/json",
			expectedBody: `{"from":"4.21.9","to":"4.21.10","fromPullSpec":"quay.io/openshift-release-dev/ocp-release:4.21.9-x86_64","toPullSpec":"quay.io/openshift-release-dev/ocp-release:4.21.10-x86_64","changelog":"Bugfixes"}`,
		},
		{
			name:         "markdown format without pullspecs",
			from:         "1.0.0",
			to:           "2.0.0",
			changelog:    "# Changelog\n\n- Feature 1",
			format:       "md",
			fromPullSpec: "",
			toPullSpec:   "",
			expectedCT:   "text/markdown",
			expectedBody: "# Changelog\n\n- Feature 1",
		},
		{
			name:         "markdown format with pullspecs",
			from:         "4.21.9",
			to:           "4.21.10",
			changelog:    "# Changelog\n\n- Feature 1",
			format:       "md",
			fromPullSpec: "quay.io/openshift-release-dev/ocp-release:4.21.9-x86_64",
			toPullSpec:   "quay.io/openshift-release-dev/ocp-release:4.21.10-x86_64",
			expectedCT:   "text/markdown",
			expectedBody: "<!-- PullSpec Information -->\n<!-- From: quay.io/openshift-release-dev/ocp-release:4.21.9-x86_64 -->\n<!-- To: quay.io/openshift-release-dev/ocp-release:4.21.10-x86_64 -->\n\n# Changelog\n\n- Feature 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			opts.writeChangelogResponse(w, tt.from, tt.to, tt.changelog, tt.format, tt.fromPullSpec, tt.toPullSpec)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, tt.expectedCT) {
				t.Errorf("Expected Content-Type to start with %q, got %q", tt.expectedCT, contentType)
			}

			body := strings.TrimSpace(w.Body.String())
			if body != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, body)
			}
		})
	}
}
