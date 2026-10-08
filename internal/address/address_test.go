package address

import "testing"

func TestCanonicalAddresses(t *testing.T) {
	for _, item := range []struct {
		protocol, host, path, want string
		port                       int32
	}{
		{"http", "auth-host", "/", "http://auth-host:81/", 81},
		{"http", "auth-host", "", "http://auth-host:81", 81},
		{"https", "127.0.0.1", "api space", "https://127.0.0.1:65535/api%20space", 65535},
		{"https", "127.0.0.1", "", "https://127.0.0.1:65535", 65535},
		{"http", "::1", "/", "http://[::1]:1/", 1},
		{"grpc", "auth-host", "/ignored", "grpc://auth-host:81", 81},
		{"tcp", "10.0.0.1", "", "tcp://10.0.0.1:81", 81},
		{"udp", "::1", "", "udp://[::1]:81", 81},
	} {
		got, err := Build(item.protocol, item.host, item.port, item.path)
		if err != nil || got != item.want {
			t.Fatalf("%+v: %q %v", item, got, err)
		}
	}
	for _, host := range []string{"http://host", "host:80", "host/path", " bad", "bad host", "[::1]", "-bad", "bad..host", "bad@host"} {
		if _, err := Build("http", host, 80, "/"); err == nil {
			t.Errorf("accepted %q", host)
		}
	}
	for _, port := range []int32{0, -1, 65536} {
		if _, err := Build("http", "host", port, "/"); err == nil {
			t.Errorf("accepted port %d", port)
		}
	}
	for _, path := range []string{"//host", "/x?q=1", "/x#fragment", "/bad\n"} {
		if _, err := Build("http", "host", 80, path); err == nil {
			t.Errorf("accepted path %q", path)
		}
	}
}

func TestNormalizeHostAcceptsConfluentCloudHostnames(t *testing.T) {
	for _, item := range []struct {
		raw  string
		want string
	}{
		{"pkc-lgk0v.us-west1.gcp.confluent.cloud", "pkc-lgk0v.us-west1.gcp.confluent.cloud"},
		{" https://pkc-lgk0v.us-west1.gcp.confluent.cloud/ ", "pkc-lgk0v.us-west1.gcp.confluent.cloud"},
		{"pkc-lgk0v.us-west1.gcp.confluent.cloud:9092", "pkc-lgk0v.us-west1.gcp.confluent.cloud"},
		{"https://pkc-lgk0v.us-west1.gcp.confluent.cloud:9092/", "pkc-lgk0v.us-west1.gcp.confluent.cloud"},
	} {
		got, err := NormalizeHost(item.raw)
		if err != nil || got != item.want {
			t.Fatalf("NormalizeHost(%q) = %q, %v; want %q", item.raw, got, err, item.want)
		}
	}
}
