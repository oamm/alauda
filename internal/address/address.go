// Package address defines server-generated endpoint presentation.
package address

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

func NormalizeHost(value string) (string, error) {
	host := strings.TrimSpace(value)
	if host == "" {
		return "", fmt.Errorf("address must be a bare DNS name or IP address without scheme, port or path")
	}
	if err := ValidateHost(host); err != nil {
		return "", err
	}
	return host, nil
}

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
func FormatValue(kind, host string, port int32, path string) (string, error) {
	if err := ValidateHost(host); err != nil {
		return "", err
	}
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("port must be between 1 and 65535")
	}
	capabilities, err := Capabilities(kind)
	if err != nil {
		return "", err
	}
	path = PublicPath(kind, path)
	if err := ValidatePath(path); err != nil {
		return "", err
	}
	if capabilities.HasURIScheme {
		if path != "" && !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		return (&url.URL{Scheme: capabilities.URIScheme, Host: net.JoinHostPort(host, strconv.Itoa(int(port))), Path: path}).String(), nil
	}
	return net.JoinHostPort(host, strconv.Itoa(int(port))), nil
}
