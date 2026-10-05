package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the registry configuration
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Storage   StorageConfig   `yaml:"storage"`
	Health    HealthConfig    `yaml:"health"`
	Telemetry TelemetryConfig `yaml:"telemetry"`
	Alerts    AlertsConfig    `yaml:"alerts"`
	Auth      AuthConfig      `yaml:"auth"`
	RateLimit RateLimitConfig `yaml:"rateLimit"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Address      string        `yaml:"address"`
	Port         int           `yaml:"port"`
	TLSCert      string        `yaml:"tlsCert"`
	TLSKey       string        `yaml:"tlsKey"`
	DevMode      bool          `yaml:"devMode"`
	LogLevel     string        `yaml:"logLevel"`
	ReadTimeout  time.Duration `yaml:"readTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Path        string        `yaml:"path"`
	BusyTimeout time.Duration `yaml:"busyTimeout"`
}

// HealthConfig represents health check configuration
type HealthConfig struct {
	WorkerCount             int           `yaml:"workerCount"`
	CheckInterval           time.Duration `yaml:"checkInterval"`
	DefaultTimeout          time.Duration `yaml:"defaultTimeout"`
	DefaultFailureThreshold int           `yaml:"defaultFailureThreshold"`
	DefaultSuccessThreshold int           `yaml:"defaultSuccessThreshold"`
	ResultRetentionHours    int           `yaml:"resultRetentionHours"`
	CheckQueueCapacity      int           `yaml:"checkQueueCapacity"`
	ResultQueueCapacity     int           `yaml:"resultQueueCapacity"`
}

// TelemetryConfig represents telemetry configuration
type TelemetryConfig struct {
	Enabled        bool   `yaml:"enabled"`
	OTLPEndpoint   string `yaml:"otlpEndpoint"`
	ServiceName    string `yaml:"serviceName"`
	ServiceVersion string `yaml:"serviceVersion"`
}

// AlertsConfig represents alerts configuration
type AlertsConfig struct {
	Enabled    bool          `yaml:"enabled"`
	Cooldown   time.Duration `yaml:"cooldown"`
	MaxRetries int           `yaml:"maxRetries"`
}

// AuthConfig controls local authentication and API token authorization.
type AuthConfig struct {
	Enabled                 bool          `yaml:"enabled"`
	BootstrapAdminUsername  string        `yaml:"bootstrapAdminUsername"`
	BootstrapAdminEmail     string        `yaml:"bootstrapAdminEmail"`
	BootstrapCredentialPath string        `yaml:"bootstrapCredentialPath"`
	TokenTTL                time.Duration `yaml:"tokenTTL"`
	SessionCookieName       string        `yaml:"sessionCookieName"`
}

// RateLimitConfig controls API request throttling.
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requestsPerMinute"`
	Burst             int  `yaml:"burst"`
}

// Load loads configuration from file or environment variables
func Load() (*Config, error) {
	cfg := defaultConfig()

	// Try to load from file if specified
	configFile := os.Getenv("REGISTRY_CONFIG")
	if configFile == "" {
		configFile = "config.yaml"
	}

	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Override with environment variables
	if port := os.Getenv("REGISTRY_PORT"); port != "" {
		if parsed, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = parsed
		}
	}
	if addr := os.Getenv("REGISTRY_ADDRESS"); addr != "" {
		cfg.Server.Address = addr
	}
	if logLevel := os.Getenv("REGISTRY_LOG_LEVEL"); logLevel != "" {
		cfg.Server.LogLevel = logLevel
	}
	if dev := os.Getenv("REGISTRY_DEV_MODE"); dev != "" {
		if parsed, err := strconv.ParseBool(dev); err == nil {
			cfg.Server.DevMode = parsed
		}
	}
	if dbPath := os.Getenv("REGISTRY_STORAGE_PATH"); dbPath != "" {
		cfg.Storage.Path = dbPath
	}
	if workers := os.Getenv("REGISTRY_HEALTH_WORKERS"); workers != "" {
		if parsed, err := strconv.Atoi(workers); err == nil && parsed > 0 {
			cfg.Health.WorkerCount = parsed
		}
	}
	if telemetry := os.Getenv("REGISTRY_TELEMETRY_ENABLED"); telemetry != "" {
		if parsed, err := strconv.ParseBool(telemetry); err == nil {
			cfg.Telemetry.Enabled = parsed
		}
	}
	if endpoint := os.Getenv("REGISTRY_OTLP_ENDPOINT"); endpoint != "" {
		cfg.Telemetry.OTLPEndpoint = endpoint
	}
	if authEnabled := os.Getenv("REGISTRY_AUTH_ENABLED"); authEnabled != "" {
		if parsed, err := strconv.ParseBool(authEnabled); err == nil {
			cfg.Auth.Enabled = parsed
		}
	}
	if username := os.Getenv("REGISTRY_BOOTSTRAP_ADMIN_USERNAME"); username != "" {
		cfg.Auth.BootstrapAdminUsername = username
	}
	if email := os.Getenv("REGISTRY_BOOTSTRAP_ADMIN_EMAIL"); email != "" {
		cfg.Auth.BootstrapAdminEmail = email
	}
	if credentialPath := os.Getenv("REGISTRY_BOOTSTRAP_CREDENTIAL_PATH"); credentialPath != "" {
		cfg.Auth.BootstrapCredentialPath = credentialPath
	}
	if cookieName := os.Getenv("REGISTRY_AUTH_SESSION_COOKIE"); cookieName != "" {
		cfg.Auth.SessionCookieName = cookieName
	}
	if ttl := os.Getenv("REGISTRY_AUTH_TOKEN_TTL"); ttl != "" {
		if parsed, err := time.ParseDuration(ttl); err == nil && parsed > 0 {
			cfg.Auth.TokenTTL = parsed
		}
	}
	if rateLimitEnabled := os.Getenv("REGISTRY_RATE_LIMIT_ENABLED"); rateLimitEnabled != "" {
		if parsed, err := strconv.ParseBool(rateLimitEnabled); err == nil {
			cfg.RateLimit.Enabled = parsed
		}
	}
	if rpm := os.Getenv("REGISTRY_RATE_LIMIT_REQUESTS_PER_MINUTE"); rpm != "" {
		if parsed, err := strconv.Atoi(rpm); err == nil && parsed > 0 {
			cfg.RateLimit.RequestsPerMinute = parsed
		}
	}
	if burst := os.Getenv("REGISTRY_RATE_LIMIT_BURST"); burst != "" {
		if parsed, err := strconv.Atoi(burst); err == nil && parsed > 0 {
			cfg.RateLimit.Burst = parsed
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if c.Server.Address == "" {
		return fmt.Errorf("server.address is required")
	}
	if c.Server.ReadTimeout <= 0 || c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("server read/write timeouts must be positive")
	}
	if c.Storage.Path == "" {
		return fmt.Errorf("storage.path is required")
	}
	if c.Health.WorkerCount <= 0 {
		return fmt.Errorf("health.workerCount must be positive")
	}
	if c.Health.CheckInterval <= 0 {
		return fmt.Errorf("health.checkInterval must be positive")
	}
	if c.Health.CheckQueueCapacity <= 0 {
		return fmt.Errorf("health.checkQueueCapacity must be positive")
	}
	if c.Health.ResultRetentionHours <= 0 {
		return fmt.Errorf("health.resultRetentionHours must be positive")
	}
	if !c.Server.DevMode && !c.Auth.Enabled {
		return fmt.Errorf("auth must be enabled outside dev mode")
	}
	if c.Auth.BootstrapAdminUsername == "" {
		return fmt.Errorf("auth.bootstrapAdminUsername is required")
	}
	if c.Auth.BootstrapCredentialPath == "" {
		return fmt.Errorf("auth.bootstrapCredentialPath is required")
	}
	if c.Auth.SessionCookieName == "" {
		return fmt.Errorf("auth.sessionCookieName is required")
	}
	if c.Auth.TokenTTL <= 0 {
		return fmt.Errorf("auth.tokenTTL must be positive")
	}
	if c.RateLimit.Enabled {
		if c.RateLimit.RequestsPerMinute <= 0 {
			return fmt.Errorf("rateLimit.requestsPerMinute must be positive")
		}
		if c.RateLimit.Burst <= 0 {
			return fmt.Errorf("rateLimit.burst must be positive")
		}
	}
	return nil
}

// defaultConfig returns the default configuration
func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Address:      "0.0.0.0",
			Port:         9700,
			LogLevel:     "info",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
		Storage: StorageConfig{
			Path:        "./data/registry.db",
			BusyTimeout: 5 * time.Second,
		},
		Health: HealthConfig{
			WorkerCount:             20,
			CheckInterval:           10 * time.Second,
			DefaultTimeout:          5 * time.Second,
			DefaultFailureThreshold: 3,
			DefaultSuccessThreshold: 2,
			ResultRetentionHours:    24,
			CheckQueueCapacity:      1000,
			ResultQueueCapacity:     1000,
		},
		Telemetry: TelemetryConfig{
			Enabled:        true,
			OTLPEndpoint:   "http://localhost:4317",
			ServiceName:    "registry",
			ServiceVersion: "0.1.0",
		},
		Alerts: AlertsConfig{
			Enabled:    true,
			Cooldown:   10 * time.Minute,
			MaxRetries: 5,
		},
		Auth: AuthConfig{
			Enabled:                 true,
			BootstrapAdminUsername:  "root",
			BootstrapAdminEmail:     "root@example.local",
			BootstrapCredentialPath: "./data/bootstrap-admin-credential",
			TokenTTL:                24 * time.Hour,
			SessionCookieName:       "alauda_session",
		},
		RateLimit: RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 600,
			Burst:             60,
		},
	}
}
