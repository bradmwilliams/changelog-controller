package releasecontroller

import "fmt"

// Architecture represents a CPU architecture supported by OpenShift
type Architecture string

const (
	ArchitectureAMD64   Architecture = "amd64"
	ArchitectureARM64   Architecture = "arm64"
	ArchitectureS390x   Architecture = "s390x"
	ArchitecturePPC64le Architecture = "ppc64le"
	ArchitectureMulti   Architecture = "multi"
)

// AllArchitectures returns all supported architectures
func AllArchitectures() []Architecture {
	return []Architecture{
		ArchitectureAMD64,
		ArchitectureARM64,
		ArchitectureS390x,
		ArchitecturePPC64le,
		ArchitectureMulti,
	}
}

// Product represents an OpenShift product variant
type Product string

const (
	ProductOCP    Product = "ocp"
	ProductOrigin Product = "origin"
)

// AllProducts returns all supported products
func AllProducts() []Product {
	return []Product{
		ProductOCP,
		ProductOrigin,
	}
}

// Endpoint represents a release-controller endpoint
type Endpoint struct {
	Architecture Architecture
	Product      Product
	URL          string
}

// NewEndpoint creates a new endpoint with validation
func NewEndpoint(arch Architecture, product Product) (*Endpoint, error) {
	// Validate that origin only supports amd64
	if product == ProductOrigin && arch != ArchitectureAMD64 {
		return nil, fmt.Errorf("product %s only supports architecture %s", ProductOrigin, ArchitectureAMD64)
	}

	url := fmt.Sprintf("https://%s.%s.releases.ci.openshift.org", arch, product)
	return &Endpoint{
		Architecture: arch,
		Product:      product,
		URL:          url,
	}, nil
}

// String returns the endpoint URL
func (e *Endpoint) String() string {
	return e.URL
}
