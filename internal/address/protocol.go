package address

import (
	"fmt"
	"strings"
)

// EndpointCapabilities describes public endpoint semantics. Endpoint kind is
// not a URI scheme unless HasURIScheme is true.
type EndpointCapabilities struct {
	SupportsPath bool
	HasURIScheme bool
	URIScheme    string
}

func Capabilities(kind string) (EndpointCapabilities, error) {
	switch strings.ToLower(strings.TrimPrefix(kind, "ENDPOINT_KIND_")) {
	case "http":
		return EndpointCapabilities{SupportsPath: true, HasURIScheme: true, URIScheme: "http"}, nil
	case "https":
		return EndpointCapabilities{SupportsPath: true, HasURIScheme: true, URIScheme: "https"}, nil
	case "tcp", "udp", "grpc", "postgres", "redis", "custom":
		return EndpointCapabilities{}, nil
	default:
		return EndpointCapabilities{}, fmt.Errorf("unsupported endpoint kind")
	}
}

func ValidateEndpointPath(kind, path string) error {
	capabilities, err := Capabilities(kind)
	if err != nil {
		return err
	}
	if !capabilities.SupportsPath && path != "" {
		return fmt.Errorf("path is only supported for HTTP and HTTPS endpoints")
	}
	return ValidatePath(path)
}

func ValidateEndpoint(kind string, port int32, path string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return ValidateEndpointPath(kind, path)
}

// PublicPath omits incompatible values from public projection.
func PublicPath(kind, path string) string {
	capabilities, err := Capabilities(kind)
	if err != nil || !capabilities.SupportsPath {
		return ""
	}
	return path
}
