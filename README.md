# Alauda Service Registry

Alauda is a self-hosted service registry and operational control plane for registering services, environments, instances, endpoints, health checks, incidents, and execution results.

## Features

- Service, environment, instance, and endpoint registration
- Health checks, execution history, incidents, and recovery workflows
- REST and ConnectRPC APIs with OpenAPI documentation
- Web console for day-to-day operations
- `alauda` CLI for automation and public management workflows
- SQLite-backed persistence with Docker support
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
  YOUR_DOCKERHUB_USERNAME/service-registry:latest
```

Open `http://localhost:9700` after the container starts. Replace `YOUR_DOCKERHUB_USERNAME` with the Docker Hub account that publishes your image.

The image stores its SQLite database under `/data`. Configuration can be supplied with the project configuration file or environment variables; see [Configuration](docs/CONFIGURATION.md) and [Docker](docs/DOCKER.md).

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
docker run --rm -p 9700:9700 -v "$(pwd)/data:/data" alauda/service-registry:local
```

Start the local development dependencies with `make dev`. Run the frontend separately with `make dev-ui` when you need the Vite development server.

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
