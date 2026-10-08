# Alauda Service Registry

Alauda is a self-hosted service registry and operational control plane for registering services, environments, instances, endpoints, health checks, incidents, and execution results.

## Features

- Service, environment, instance, and endpoint registration
- Health checks, execution history, incidents, and recovery workflows
- REST and ConnectRPC APIs with OpenAPI documentation
- Web console for day-to-day operations
- `alauda` CLI for automation and public management workflows
- PostgreSQL and SQLite-backed persistence
- Runtime registration and service discovery workflows

## Quick Start With Docker

Pull the latest public image:

```sh
docker pull YOUR_DOCKERHUB_USERNAME/service-registry:latest
```

Run the registry with a persistent data directory:

```sh
docker run --name alauda-registry \
  -p 9700:9700 \
  -v "$(pwd)/data:/data" \
  -e ALAUDA_STORAGE_PROVIDER=sqlite \
  -e ALAUDA_DATABASE_URL=/data/alauda.db \
  YOUR_DOCKERHUB_USERNAME/service-registry:latest
```

Open `http://localhost:9700` after the container starts. Replace `YOUR_DOCKERHUB_USERNAME` with the Docker Hub account that publishes your image.

The default configuration uses PostgreSQL. For a single-node SQLite deployment, mount `/data` and set `ALAUDA_STORAGE_PROVIDER=sqlite` and `ALAUDA_DATABASE_URL=/data/alauda.db`. See [Storage Providers](docs/STORAGE.md), [Configuration](docs/CONFIGURATION.md), and [Docker](docs/DOCKER.md).

## Build From Source

Requirements:

- Go 1.25 or newer
- Node.js 22 or newer
- npm
- Docker, when building or running the container image

Install dependencies and run the test suites:

```sh
make setup
make test
cd web && npm test -- --run
```

Build the server and CLI:

```sh
make build
make build-alauda
```

Build and run a local image:

```sh
make docker-build IMAGE=alauda/service-registry:local
docker run --rm -p 9700:9700 -v "$(pwd)/data:/data" -e ALAUDA_STORAGE_PROVIDER=sqlite -e ALAUDA_DATABASE_URL=/data/alauda.db alauda/service-registry:local
```

Start the recommended PostgreSQL development workflow with `./scripts/dev.sh` or `make dev`. To use a local SQLite file instead, run `./scripts/dev.sh --db sqlite`. Run the frontend separately with `make dev-ui` when you need the Vite development server.

## Local Development Tutorial

Install dependencies once:

```sh
make setup
```

Start the full development app with PostgreSQL:

```sh
make dev
```

Or run the helper script directly:

```sh
./scripts/dev.sh
```

On Windows PowerShell, use the underlying script:

```powershell
.\scripts\run-app.ps1 -DatabaseProvider postgres
```

The script starts PostgreSQL with Docker Compose, applies migrations, builds the embedded UI, and serves Alauda at `http://127.0.0.1:9700`. On the first run it creates the `root` administrator and prints a temporary password. Sign in with that temporary password, then change it when prompted.

To use SQLite instead of PostgreSQL:

```sh
./scripts/dev.sh --db sqlite
```

Reset the local development initial password when you lose it or want to rotate it:

```sh
./scripts/dev.sh --reset-dev-password
```

On Windows PowerShell:

```powershell
.\scripts\run-app.ps1 -DatabaseProvider postgres -ResetDevPassword
```

The reset keeps the existing development database and records, revokes existing sessions, writes a new temporary password to the bootstrap credential file, and prints it in the terminal. After signing in, change the password again.

## API and Documentation

- [API overview](docs/API.md)
- [Public API](docs/PUBLIC_API.md)
- [UI API reference](docs/UI_API.md)
- [CLI guide](docs/CLI.md)
- [Configuration](docs/CONFIGURATION.md)
- [Docker usage](docs/DOCKER.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Service discovery](docs/SERVICE_DISCOVERY.md)
- [Health model](docs/HEALTH.md)

The web console is served by the registry binary. The default server port is `9700`; use `registry server --help` or the configuration documentation for available options.

## Docker Hub Publishing

The workflow in `.github/workflows/docker-publish.yml` publishes the production image to Docker Hub:

- pushes to `main` publish the `latest` tag
- version tags such as `v1.2.3` publish semver tags and the corresponding version tag
- manual runs publish the branch-derived tag

Configure these GitHub repository secrets before enabling publication:

- `DOCKERHUB_USERNAME`: Docker Hub account name
- `DOCKERHUB_TOKEN`: Docker Hub access token with permission to push to the repository

The workflow assumes the Docker Hub repository is named `service-registry`. Change `IMAGE_REPOSITORY` in the workflow if the public repository uses another name.

## Contributing

Read the project documentation and run the Go and frontend test suites before opening a pull request. Keep API contracts and migration behavior documented when changing server or UI workflows.

## License

Alauda is licensed under the [Apache License 2.0](LICENSE).
