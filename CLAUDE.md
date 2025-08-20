# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Docker Registry proxy that integrates with HashiCorp Vault for credential management. It implements the Docker Registry v2 API and transparently forwards requests to actual registries using credentials retrieved from Vault.

## Development Commands

### Build and Test
- `go build -o vault-docker-proxy` - Build the binary
- `go test ./...` - Run all tests
- `go mod tidy` - Clean up dependencies

### Local Development with Docker Compose
- `cd docker && docker-compose up -d` - Start Vault and proxy for testing
- `docker-compose down` - Stop all services
- `docker-compose logs vault-docker-proxy` - View proxy logs

### Manual Testing
- `./vault-docker-proxy` - Run proxy locally (requires Vault at localhost:8200)

## Architecture

The project follows a clean architecture pattern:

- `main.go` - Application entry point and HTTP server setup
- `pkg/auth/` - Authentication configuration parsing and middleware
- `pkg/vault/` - HashiCorp Vault client integration
- `pkg/cache/` - Credential caching with TTL (5-minute default)
- `pkg/registry/` - Docker Registry v2 API proxy logic
- `docker/` - Docker Compose setup and Dockerfile

### Key Components

**Authentication Flow:**
1. Client sends username in format `<registry_type>;<vault_path>;<registry_url>`
2. Password field contains Vault authentication token
3. Proxy extracts configuration, retrieves credentials from Vault
4. Credentials are cached with 5-minute TTL
5. Requests forwarded to actual registry with real credentials

**Supported Registry Types:**
- `docker` - Standard Docker registries (Docker Hub, private registries)
- `ecr` - AWS Elastic Container Registry (planned)
- `gcr` - Google Container Registry (planned)

## Configuration

Environment variables:
- `PORT` - Proxy server port (default: 8080)
- `VAULT_ADDR` - Vault server address (default: http://localhost:8200)

## Testing with Aqua Security

Configure Aqua registry as:
- **Registry URL**: `http://localhost:8080`
- **Username**: `docker;docker-hub;registry.hub.docker.com`
- **Password**: Vault token (e.g., `dev-root-token` in dev mode)

## Common Vault Operations

Store credentials:
```bash
vault kv put secret/docker-hub username="user" password="pass"
```

Retrieve credentials:
```bash
vault kv get secret/docker-hub
```