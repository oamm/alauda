package address

import "testing"

func TestEndpointCapabilities(t *testing.T) {
	for _, protocol := range []string{"http", "https", "tcp", "udp", "grpc"} {
		t.Run(protocol, func(t *testing.T) {
			capabilities, err := Capabilities(protocol)
			if err != nil {
				t.Fatal(err)
			}
			if capabilities.SupportsPath != (protocol == "http" || protocol == "https") {
				t.Fatal("incorrect capabilities")
			}
			if err := ValidateEndpoint(protocol, 81, ""); err != nil {
				t.Fatal(err)
			}
			err = ValidateEndpoint(protocol, 81, "/api")
			if (err == nil) != capabilities.SupportsPath {
				t.Fatalf("path validation: %v", err)
			}
			if !capabilities.SupportsPath && PublicPath(protocol, "/legacy") != "" {
				t.Fatal("legacy path exposed")
			}
			for _, port := range []int32{0, -1, 65536} {
				if ValidateEndpoint(protocol, port, "") == nil {
					t.Fatal("invalid port accepted")
				}
			}
		})
	}
	if ValidateEndpoint("unsupported", 81, "") == nil {
		t.Fatal("unknown protocol accepted")
	}
}
