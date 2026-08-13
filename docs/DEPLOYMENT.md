# Deployment Guide

## Production Startup

```powershell
$env:REGISTRY_CONFIG="config.production.yaml"
registry server
```

Use `server.devMode: false` and enable `auth.enabled` for production. Configure either a bootstrap admin password for interactive login or a fixed bootstrap token for automation.

## Health And Metrics

| Endpoint   | Purpose                 |
| ---------- | ----------------------- |
| `/healthz` | Process liveness        |
| `/readyz`  | Database readiness      |
| `/version` | Binary version metadata |
| `/metrics` | Prometheus text metrics |

## SQLite Backups

Run SQLite in WAL mode, which the server enables on startup. Back up the database with the SQLite online backup API where possible. If filesystem backups are used, copy the `.db`, `.db-wal`, and `.db-shm` files together while the server is stopped or quiesced.

Recommended schedule:

- Hourly backups for high-change environments.
- Daily retained backups for 30 days.
- Monthly retained backups for 12 months.
- Restore tests at least quarterly.

## Retention

Detailed health results are pruned on startup using `health.resultRetentionHours`. Audit logs are retained indefinitely. Events, incidents, and availability history are retained for operational history and should be archived externally before manual cleanup.

## Security Checklist

- Enable `auth.enabled`.
- Keep `server.devMode` disabled.
- Set a strong bootstrap password or bootstrap token.
- Set `rateLimit.enabled`.
- Store config secrets outside source control.
- Scrape `/metrics` and alert on `/readyz` failures.
