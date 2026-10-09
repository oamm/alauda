package address

import (
	"fmt"
	"strings"
)

// ProtocolCapabilities describes the existing public endpoint representations.
type ProtocolCapabilities struct{ SupportsPath bool }

func Capabilities(protocol string) (ProtocolCapabilities, error) {
	switch strings.ToLower(strings.TrimPrefix(protocol, "PROTOCOL_")) {
	case "http", "https":
		return ProtocolCapabilities{SupportsPath: true}, nil
	case "tcp", "udp", "grpc":
		return ProtocolCapabilities{}, nil
	default:
		return ProtocolCapabilities{}, fmt.Errorf("unsupported endpoint protocol")
	}
}

func ValidateEndpointPath(protocol, path string) error {
	capabilities, err := Capabilities(protocol)
	if err != nil {
		return err
	}
	if !capabilities.SupportsPath && path != "" {
		return fmt.Errorf("path is only supported for HTTP and HTTPS endpoints")
	}
	return ValidatePath(path)
}

func ValidateEndpoint(protocol string, port int32, path string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return ValidateEndpointPath(protocol, path)
}

// PublicPath omits incompatible legacy values without rewriting stored records.
func PublicPath(protocol, path string) string {
	capabilities, err := Capabilities(protocol)
	if err != nil || !capabilities.SupportsPath {
		return ""
	}
	return path
}
