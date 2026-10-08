# Development Guide

This guide covers building Alauda from source, running the local development environment, and validating changes.

## Requirements

- Go 1.25 or newer
- Node.js 22 or newer
- npm
- Docker, for PostgreSQL and container builds

## Setup

Install Go and frontend dependencies:

```sh
make setup
```

Generate API code and build the server or CLI:

```sh
make proto
make build
make build-alauda
```

## Local Development

Start the PostgreSQL-backed development environment:

```sh
make dev
```

Or run the helper directly:

```sh
./scripts/dev.sh
```

On Windows PowerShell:

```powershell
.\scripts\run-app.ps1 -DatabaseProvider postgres
```

The development helper starts PostgreSQL with Docker Compose, applies migrations, builds the embedded UI, and serves Alauda at `http://127.0.0.1:9700`. On first startup it creates the `root` administrator and prints a temporary password. Change that password after signing in.

Use SQLite for a local file-based development database:

```sh
./scripts/dev.sh --db sqlite
```

Reset the local development password when needed:

```sh
./scripts/dev.sh --reset-dev-password
```

On Windows PowerShell:

```powershell
.\scripts\run-app.ps1 -DatabaseProvider postgres -ResetDevPassword
```

## Docker Compose

The repository Compose file starts the PostgreSQL-backed development stack:

```sh
docker compose up --build
```

The web console is available at `http://127.0.0.1:9700`. Stop the stack with:

```sh
docker compose down
```

Add `-v` only when you intend to remove the local PostgreSQL and registry volumes.

## Local Image

Build and run a local SQLite-backed image:

```sh
make docker-build IMAGE=alauda:local
docker run --rm -p 9700:9700 -v "$(pwd)/data:/data" \
  -e ALAUDA_STORAGE_PROVIDER=sqlite \
  -e ALAUDA_DATABASE_URL=/data/alauda.db \
  alauda:local
```

For PostgreSQL, set `ALAUDA_STORAGE_PROVIDER=postgres` and `ALAUDA_DATABASE_URL` to a reachable database URL. See [Docker usage](DOCKER.md) for the production image details.

## Tests and Quality

Run the Go suite:

```sh
make test
```

Run frontend tests and the production build:

```sh
cd web
npm test -- --run
npm run build
```

Run formatting and static checks when changing code:

```sh
make fmt
make vet
```

Keep API contracts, migrations, and user-facing workflows documented when changing behavior.
