package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateRejectsInvalidPort(t *testing.T) {
	cfg := defaultConfig()
	cfg.Server.Port = 70000
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected invalid port error")
	}
}

func TestValidateRequiresAuthBootstrapOutsideDev(t *testing.T) {
	cfg := defaultConfig()
	cfg.Auth.BootstrapCredentialPath = ""
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected bootstrap credential path error")
	}
}

func TestValidateRejectsDefaultConfigWithoutBootstrap(t *testing.T) {
	cfg := defaultConfig()
	cfg.Auth.SessionCookieName = ""
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected session cookie name error")
	}
}

func TestValidateRejectsDisabledAuthOutsideDev(t *testing.T) {
	cfg := defaultConfig()
	cfg.Auth.Enabled = false
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected disabled auth to be rejected outside dev mode")
	}
}

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	t.Setenv("REGISTRY_CONFIG", filepath.Join(t.TempDir(), "missing.yaml"))
	t.Setenv("REGISTRY_PORT", "9800")
	t.Setenv("REGISTRY_ADDRESS", "127.0.0.1")
	t.Setenv("REGISTRY_LOG_LEVEL", "debug")
	t.Setenv("REGISTRY_DEV_MODE", "true")
	t.Setenv("REGISTRY_STORAGE_PATH", ":memory:")
	t.Setenv("REGISTRY_HEALTH_WORKERS", "7")
	t.Setenv("REGISTRY_TELEMETRY_ENABLED", "false")
	t.Setenv("REGISTRY_OTLP_ENDPOINT", "http://collector:4317")
	t.Setenv("REGISTRY_AUTH_ENABLED", "true")
	t.Setenv("REGISTRY_BOOTSTRAP_ADMIN_USERNAME", "root")
	t.Setenv("REGISTRY_BOOTSTRAP_ADMIN_EMAIL", "root@example.test")
	t.Setenv("REGISTRY_BOOTSTRAP_CREDENTIAL_PATH", "./tmp/bootstrap")
	t.Setenv("REGISTRY_AUTH_SESSION_COOKIE", "test_session")
	t.Setenv("REGISTRY_AUTH_TOKEN_TTL", "2h")
	t.Setenv("REGISTRY_RATE_LIMIT_ENABLED", "true")
	t.Setenv("REGISTRY_RATE_LIMIT_REQUESTS_PER_MINUTE", "120")
	t.Setenv("REGISTRY_RATE_LIMIT_BURST", "12")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 9800 || cfg.Server.Address != "127.0.0.1" || cfg.Server.LogLevel != "debug" || !cfg.Server.DevMode {
		t.Fatalf("server env overrides not applied: %+v", cfg.Server)
	}
	if cfg.Storage.Path != ":memory:" || cfg.Health.WorkerCount != 7 {
		t.Fatalf("storage/health env overrides not applied: storage=%+v health=%+v", cfg.Storage, cfg.Health)
	}
	if cfg.Telemetry.Enabled || cfg.Telemetry.OTLPEndpoint != "http://collector:4317" {
		t.Fatalf("telemetry env overrides not applied: %+v", cfg.Telemetry)
	}
	if !cfg.Auth.Enabled || cfg.Auth.BootstrapAdminUsername != "root" || cfg.Auth.TokenTTL != 2*time.Hour || cfg.Auth.BootstrapCredentialPath != "./tmp/bootstrap" || cfg.Auth.SessionCookieName != "test_session" {
		t.Fatalf("auth env overrides not applied: %+v", cfg.Auth)
	}
	if !cfg.RateLimit.Enabled || cfg.RateLimit.RequestsPerMinute != 120 || cfg.RateLimit.Burst != 12 {
		t.Fatalf("rate limit env overrides not applied: %+v", cfg.RateLimit)
	}
}

func TestLoadReadsYAMLConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`
server:
  address: 127.0.0.1
  port: 9900
  devMode: true
  readTimeout: 3s
  writeTimeout: 4s
storage:
  path: ":memory:"
health:
  workerCount: 3
  checkInterval: 2s
  checkQueueCapacity: 10
  resultRetentionHours: 12
auth:
  tokenTTL: 30m
rateLimit:
  enabled: true
  requestsPerMinute: 60
  burst: 6
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("REGISTRY_CONFIG", path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Port != 9900 || cfg.Server.ReadTimeout != 3*time.Second || cfg.Storage.Path != ":memory:" {
		t.Fatalf("yaml config not applied: %+v", cfg)
	}
}

func TestValidateRejectsInvalidCoreSettings(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "empty address", mutate: func(c *Config) { c.Server.Address = "" }},
		{name: "nonpositive timeouts", mutate: func(c *Config) { c.Server.ReadTimeout = 0 }},
		{name: "empty storage path", mutate: func(c *Config) { c.Storage.Path = "" }},
		{name: "nonpositive workers", mutate: func(c *Config) { c.Health.WorkerCount = 0 }},
		{name: "nonpositive interval", mutate: func(c *Config) { c.Health.CheckInterval = 0 }},
		{name: "nonpositive queue", mutate: func(c *Config) { c.Health.CheckQueueCapacity = 0 }},
		{name: "nonpositive retention", mutate: func(c *Config) { c.Health.ResultRetentionHours = 0 }},
		{name: "nonpositive token ttl", mutate: func(c *Config) { c.Auth.TokenTTL = 0 }},
		{name: "nonpositive rate limit rpm", mutate: func(c *Config) {
			c.RateLimit.Enabled = true
			c.RateLimit.RequestsPerMinute = 0
		}},
		{name: "nonpositive rate limit burst", mutate: func(c *Config) {
			c.RateLimit.Enabled = true
			c.RateLimit.Burst = 0
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := defaultConfig()
			tc.mutate(cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}
