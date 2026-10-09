package address

import "testing"

func TestEndpointCapabilities(t *testing.T) {
	for _, kind := range []string{"http", "https", "tcp", "udp", "grpc", "postgres", "redis", "custom"} {
		t.Run(kind, func(t *testing.T) {
			capabilities, err := Capabilities(kind)
			if err != nil {
				t.Fatal(err)
			}
			if capabilities.SupportsPath != (kind == "http" || kind == "https") {
				t.Fatal("incorrect capabilities")
			}
			if err := ValidateEndpoint(kind, 81, ""); err != nil {
				t.Fatal(err)
			}
			err = ValidateEndpoint(kind, 81, "/api")
			if (err == nil) != capabilities.SupportsPath {
				t.Fatalf("path validation: %v", err)
			}
			if !capabilities.SupportsPath && PublicPath(kind, "/legacy") != "" {
				t.Fatal("legacy path exposed")
			}
			for _, port := range []int32{0, -1, 65536} {
				if ValidateEndpoint(kind, port, "") == nil {
					t.Fatal("invalid port accepted")
				}
			}
		})
	}
	if ValidateEndpoint("unsupported", 81, "") == nil {
		t.Fatal("unknown kind accepted")
	}
}
