# Configuration Reference

Configuration precedence is:

1. CLI flags for `registry`
2. Environment variables
3. YAML file from `REGISTRY_CONFIG` or `config.yaml`
4. Built-in defaults

## Core Settings

```yaml
server:
  address: "0.0.0.0"
  port: 9700
  devMode: false
  logLevel: "info"
  readTimeout: 15s
  writeTimeout: 15s

storage:
  path: "./data/registry.db"
  busyTimeout: 5s

auth:
  enabled: true
  bootstrapAdminUsername: "admin"
  bootstrapAdminEmail: "admin@example.local"
  bootstrapAdminPassword: ""
  bootstrapAdminTokenName: "bootstrap-admin"
  bootstrapAdminToken: ""
  tokenTTL: 24h

rateLimit:
  enabled: true
  requestsPerMinute: 600
  burst: 60
```

## Environment Variables

| Variable                                  | Description                                |
| ----------------------------------------- | ------------------------------------------ |
| `REGISTRY_CONFIG`                         | YAML config file path                      |
| `REGISTRY_ADDRESS`                        | HTTP bind address                          |
| `REGISTRY_PORT`                           | HTTP bind port                             |
| `REGISTRY_DEV_MODE`                       | Enables development bypasses               |
| `REGISTRY_STORAGE_PATH`                   | SQLite database path                       |
| `REGISTRY_HEALTH_WORKERS`                 | Health worker count                        |
| `REGISTRY_TELEMETRY_ENABLED`              | Enable OpenTelemetry provider              |
| `REGISTRY_OTLP_ENDPOINT`                  | OTLP endpoint reserved for exporter wiring |
| `REGISTRY_AUTH_ENABLED`                   | Require auth for API routes                |
| `REGISTRY_BOOTSTRAP_ADMIN_USERNAME`       | Bootstrap admin username                   |
| `REGISTRY_BOOTSTRAP_ADMIN_EMAIL`          | Bootstrap admin email                      |
| `REGISTRY_BOOTSTRAP_ADMIN_PASSWORD`       | Bootstrap admin password                   |
| `REGISTRY_BOOTSTRAP_ADMIN_TOKEN`          | Fixed bootstrap API token                  |
| `REGISTRY_AUTH_TOKEN_TTL`                 | Login token TTL, for example `24h`         |
| `REGISTRY_RATE_LIMIT_ENABLED`             | Enable API rate limiting                   |
| `REGISTRY_RATE_LIMIT_REQUESTS_PER_MINUTE` | Per-client refill rate                     |
| `REGISTRY_RATE_LIMIT_BURST`               | Per-client burst capacity                  |

## Validation

The server validates port ranges, required storage paths, positive health scheduler values, token TTL, rate-limit values, and auth bootstrap settings before startup.
