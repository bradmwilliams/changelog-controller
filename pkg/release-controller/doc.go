// Package releasecontroller provides a client for interacting with OpenShift release-controller endpoints.
//
// The release-controller manages OpenShift releases across different architectures and products.
// This package provides a type-safe, validated client for accessing these endpoints.
//
// # Endpoints
//
// Release-controller endpoints follow the pattern:
//
//	https://<ARCHITECTURE>.<PRODUCT>.releases.ci.openshift.org/
//
// Where ARCHITECTURE can be: amd64, arm64, s390x, ppc64le, or multi
// And PRODUCT can be: ocp or origin
//
// Note: The origin product only supports the amd64 architecture.
//
// # Basic Usage
//
// Create a client for a specific endpoint:
//
//	client, err := releasecontroller.NewClientForEndpoint(
//	    releasecontroller.ArchitectureAMD64,
//	    releasecontroller.ProductOCP,
//	    releasecontroller.DefaultClientConfig(),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	ctx := context.Background()
//	body, err := client.Get(ctx, "/api/v1/releasestreams")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer body.Close()
//
// # Multiple Endpoints
//
// Use MultiClient to manage multiple endpoints:
//
//	mc, err := releasecontroller.NewMultiClient(releasecontroller.DefaultClientConfig())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer mc.Close()
//
//	// Access specific clients
//	client, err := mc.GetClient(releasecontroller.ProductOCP, releasecontroller.ArchitectureAMD64)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # JSON Responses
//
// Unmarshal JSON responses directly:
//
//	var release struct {
//	    Name    string `json:"name"`
//	    Version string `json:"version"`
//	}
//	err := client.GetJSON(ctx, "/api/v1/release/4.14.0", &release)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Error Handling
//
// The client automatically validates endpoint combinations and returns descriptive errors:
//
//	// This returns an error - origin only supports amd64
//	_, err := releasecontroller.NewEndpoint(releasecontroller.ArchitectureARM64, releasecontroller.ProductOrigin)
//	// Error: product origin only supports architecture amd64
//
// # Context Support
//
// All client methods accept a context.Context for cancellation and timeout control:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	body, err := client.Get(ctx, "/api/v1/releasestreams")
package releasecontroller
