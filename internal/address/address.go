// Package address defines server-generated endpoint addresses. grpc is a
// logical cleartext gRPC authority, not an HTTP URL or a TLS configuration.
package address

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

func ValidateHost(host string) error {
	if host == "" || strings.TrimSpace(host) != host || strings.ContainsAny(host, "/\\?#@[]") || strings.Contains(host, "://") {
		return fmt.Errorf("address must be a bare DNS name or IP address without scheme, port or path")
	}
	if net.ParseIP(host) != nil {
		return nil
	}
	if strings.Contains(host, ":") || len(host) > 253 {
		return fmt.Errorf("address must be a bare DNS name or IP address")
	}
	for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return fmt.Errorf("invalid DNS hostname")
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return fmt.Errorf("invalid DNS hostname")
			}
		}
	}
	return nil
}
func ValidatePath(path string) error {
	if strings.ContainsAny(path, "?#\\") || strings.HasPrefix(path, "//") {
		return fmt.Errorf("path must be a path without query, fragment or authority")
	}
	for _, c := range path {
		if unicode.IsControl(c) {
			return fmt.Errorf("path contains control characters")
		}
	}
	return nil
}
func Build(protocol, host string, port int32, path string) (string, error) {
	if err := ValidateHost(host); err != nil {
		return "", err
	}
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("port must be between 1 and 65535")
	}
	if err := ValidatePath(path); err != nil {
		return "", err
	}
	protocol = strings.ToLower(protocol)
	switch protocol {
	case "http", "https":
		if path == "" {
			path = "/"
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
	case "grpc", "tcp", "udp":
		path = ""
	default:
		return "", fmt.Errorf("unsupported protocol")
	}
	return (&url.URL{Scheme: protocol, Host: net.JoinHostPort(host, strconv.Itoa(int(port))), Path: path}).String(), nil
}
