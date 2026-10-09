package address

import "testing"

func TestCanonicalAddresses(t *testing.T) {
	for _, item := range []struct {
		kind, host, path, want string
		port                   int32
	}{
		{"http", "auth-host", "/", "http://auth-host:81/", 81},
		{"http", "auth-host", "", "http://auth-host:81", 81},
		{"https", "127.0.0.1", "api space", "https://127.0.0.1:65535/api%20space", 65535},
		{"https", "127.0.0.1", "", "https://127.0.0.1:65535", 65535},
		{"http", "::1", "/", "http://[::1]:1/", 1},
		{"grpc", "auth-host", "/ignored", "auth-host:81", 81},
		{"postgres", "10.0.0.1", "", "10.0.0.1:5432", 5432},
		{"redis", "redis.internal", "", "redis.internal:6379", 6379},
		{"custom", "pkc-lgk0v.us-west1.gcp.confluent.cloud", "", "pkc-lgk0v.us-west1.gcp.confluent.cloud:9092", 9092},
		{"tcp", "10.0.0.1", "", "10.0.0.1:81", 81},
		{"udp", "::1", "", "[::1]:81", 81},
	} {
		got, err := FormatValue(item.kind, item.host, item.port, item.path)
		if err != nil || got != item.want {
			t.Fatalf("%+v: %q %v", item, got, err)
		}
	}
	for _, host := range []string{"http://host", "host:80", "host/path", " bad", "bad host", "[::1]", "-bad", "bad..host", "bad@host"} {
		if _, err := FormatValue("http", host, 80, "/"); err == nil {
			t.Errorf("accepted %q", host)
		}
	}
	for _, port := range []int32{0, -1, 65536} {
		if _, err := FormatValue("http", "host", port, "/"); err == nil {
			t.Errorf("accepted port %d", port)
		}
	}
	for _, path := range []string{"//host", "/x?q=1", "/x#fragment", "/bad\n"} {
		if _, err := FormatValue("http", "host", 80, path); err == nil {
			t.Errorf("accepted path %q", path)
		}
	}
}

func TestNormalizeHostRequiresBareHost(t *testing.T) {
	got, err := NormalizeHost("pkc-lgk0v.us-west1.gcp.confluent.cloud")
	if err != nil || got != "pkc-lgk0v.us-west1.gcp.confluent.cloud" {
		t.Fatalf("NormalizeHost bare DNS = %q, %v", got, err)
	}
	for _, raw := range []string{
		" https://pkc-lgk0v.us-west1.gcp.confluent.cloud/ ",
		"pkc-lgk0v.us-west1.gcp.confluent.cloud:9092",
		"https://pkc-lgk0v.us-west1.gcp.confluent.cloud:9092/",
	} {
		if _, err := NormalizeHost(raw); err == nil {
			t.Fatalf("NormalizeHost(%q) accepted scheme/port/path", raw)
		}
	}
}
