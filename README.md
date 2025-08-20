# Vault Docker Registry Proxy

A proxy server that implements the Docker Registry v2 API while retrieving authentication credentials from HashiCorp Vault. This allows secure credential management for container registry access without storing credentials in configuration files.

## Architecture

The proxy acts as a transparent intermediary between Docker clients (like Aqua Security) and Docker registries:

1. Client connects to proxy with special username format
2. Proxy extracts Vault configuration from username
3. Proxy retrieves real registry credentials from Vault
4. Proxy forwards requests to actual registry with real credentials
5. Responses are transparently passed back to client

## Username Format

The username field encodes the registry configuration:
```
<registry_type>;<vault_path>;<registry_url>
```

Examples:
- `docker;docker-hub;registry.hub.docker.com` - Docker Hub via secret/docker-hub
- `docker;private-registry;myregistry.com` - Private registry via secret/private-registry
- `ecr;aws-ecr;123456789.dkr.ecr.us-east-1.amazonaws.com` - AWS ECR via secret/aws-ecr

The password field should contain the Vault authentication token.

## Quick Start

### Using Docker Compose (Recommended for Testing)

1. Start the services:
```bash
cd docker
docker-compose up -d
```

2. The proxy will be available at `http://localhost:8080`
3. Vault will be available at `http://localhost:8200` with root token `dev-root-token`

### Manual Setup

1. Start Vault server:
```bash
vault server -dev -dev-listen-address=0.0.0.0:8200
```

2. Store credentials in Vault:
```bash
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="your-vault-token"

vault kv put secret/docker-hub \
  username="your-docker-username" \
  password="your-docker-password"
```

3. Build and run the proxy:
```bash
go build -o vault-docker-proxy
./vault-docker-proxy
```

## Configuration

Environment variables:
- `PORT` - Proxy server port (default: 8080)
- `VAULT_ADDR` - Vault server address (default: http://localhost:8200)

## Usage Examples

### Testing with curl

Check API version:
```bash
curl -u "docker;docker-hub;registry.hub.docker.com:dev-root-token" \
  http://localhost:8080/v2/
```

List repositories:
```bash
curl -u "docker;docker-hub;registry.hub.docker.com:dev-root-token" \
  http://localhost:8080/v2/_catalog
```

### Integrating with Aqua Security

Configure Aqua to use the proxy as a Docker registry:

1. **Registry URL**: `http://your-proxy-server:8080`
2. **Username**: `docker;docker-hub;registry.hub.docker.com` (or your format)
3. **Password**: Your Vault authentication token

### Supported Registry Types

- `docker` - Standard Docker registries (Docker Hub, private registries)
- `ecr` - AWS Elastic Container Registry (planned)
- `gcr` - Google Container Registry (planned)

## Security Considerations

1. **Vault Token Security**: The Vault token is passed as the password field. Ensure secure token management.
2. **TLS/HTTPS**: In production, use HTTPS for all communications.
3. **Token Rotation**: Implement regular Vault token rotation.
4. **Network Security**: Secure network access between proxy, Vault, and registries.

## Development

### Project Structure
```
├── main.go                 # Main application entry point
├── pkg/
│   ├── auth/              # Authentication and configuration parsing
│   ├── cache/             # Credential caching with TTL
│   ├── registry/          # Docker Registry v2 API proxy logic
│   └── vault/             # Vault client integration
├── docker/                # Docker Compose and deployment files
└── README.md
```

### Building from Source

```bash
go build -o vault-docker-proxy
```

### Running Tests

```bash
go test ./...
```

## Troubleshooting

### Common Issues

1. **Authentication Failed**: Check Vault token and ensure credentials exist at specified path
2. **Registry Unreachable**: Verify registry URL format and network connectivity
3. **Invalid Username Format**: Ensure username follows `<type>:<vault_path>:<registry_url>` format

### Debugging

Enable debug logging:
```bash
export LOG_LEVEL=debug
./vault-docker-proxy
```

### Vault Credential Verification

Verify credentials are stored correctly:
```bash
vault kv get secret/docker-hub
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License.