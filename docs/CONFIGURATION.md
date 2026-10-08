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
  provider: postgres
  connectionString: "postgres://alauda:change-me@localhost:5432/alauda?sslmode=require"
  path: "./data/alauda.db"
  busyTimeout: 5s

auth:
  enabled: true
  bootstrapAdminUsername: "root"
  bootstrapAdminEmail: "root@example.local"
  bootstrapCredentialPath: "./data/bootstrap-admin-credential"
  sessionCookieName: "alauda_session"
  tokenTTL: 24h

rateLimit:
  enabled: true
  requestsPerMinute: 600
  burst: 60
```

## Environment Variables

| Variable                                  | Description                                   |
| ----------------------------------------- | --------------------------------------------- |
| `REGISTRY_CONFIG`                         | YAML config file path                         |
| `REGISTRY_ADDRESS`                        | HTTP bind address                             |
| `REGISTRY_PORT`                           | HTTP bind port                                |
| `REGISTRY_DEV_MODE`                       | Enables development bypasses                  |
| `REGISTRY_STORAGE_PATH`                   | SQLite database path                          |
| `ALAUDA_STORAGE_PROVIDER`                 | `postgres` (default) or `sqlite`              |
| `ALAUDA_DATABASE_URL`                     | PostgreSQL URL or SQLite database file        |
| `ALAUDA_POSTGRES_HOST`                    | PostgreSQL host when building the URL from components |
| `ALAUDA_POSTGRES_PORT`                    | PostgreSQL port when building the URL from components |
| `ALAUDA_POSTGRES_USER`                    | PostgreSQL user when building the URL from components |
| `ALAUDA_POSTGRES_PASSWORD`                | PostgreSQL password when building the URL from components |
| `ALAUDA_POSTGRES_DATABASE`                | PostgreSQL database when building the URL from components |
| `ALAUDA_POSTGRES_SSLMODE`                 | PostgreSQL sslmode when building the URL from components |
| `REGISTRY_HEALTH_WORKERS`                 | Health worker count                           |
| `REGISTRY_TELEMETRY_ENABLED`              | Enable OpenTelemetry provider                 |
| `REGISTRY_OTLP_ENDPOINT`                  | OTLP endpoint reserved for exporter wiring    |
| `REGISTRY_AUTH_ENABLED`                   | Require auth for API routes                   |
| `REGISTRY_BOOTSTRAP_ADMIN_USERNAME`       | Bootstrap admin username                      |
| `REGISTRY_BOOTSTRAP_ADMIN_EMAIL`          | Bootstrap admin email                         |
| `REGISTRY_BOOTSTRAP_CREDENTIAL_PATH`      | One-time restricted bootstrap credential file |
| `REGISTRY_AUTH_SESSION_COOKIE`            | Browser session cookie name                   |
| `REGISTRY_AUTH_TOKEN_TTL`                 | Login token TTL, for example `24h`            |
| `REGISTRY_RATE_LIMIT_ENABLED`             | Enable API rate limiting                      |
| `REGISTRY_RATE_LIMIT_REQUESTS_PER_MINUTE` | Per-client refill rate                        |
| `REGISTRY_RATE_LIMIT_BURST`               | Per-client burst capacity                     |

## Validation

The server validates the storage provider, positive health scheduler values, token TTL, rate-limit values, and security bootstrap settings before startup. Production authentication cannot be disabled. See [Storage Providers](STORAGE.md) for development and deployment guidance.
