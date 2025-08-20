#!/bin/sh

echo "Initializing Vault with test credentials..."

# Wait for Vault to be ready
sleep 5

# Enable KV v2 secrets engine at 'secret' path (usually enabled by default in dev mode)
vault secrets enable -path=secret kv-v2 || echo "KV v2 already enabled"

# Create test credentials for Docker Hub
vault kv put secret/docker-hub \
  username="your-docker-username" \
  password="your-docker-password" \
  email="your-email@example.com"

echo "Created test Docker Hub credentials at secret/docker-hub"

# Create test credentials for a private registry
vault kv put secret/private-registry \
  username="registry-user" \
  password="registry-password"

echo "Created test private registry credentials at secret/private-registry"

# Create test credentials for AWS ECR (example)
vault kv put secret/aws-ecr \
  username="AWS" \
  password="your-ecr-token"

echo "Created test AWS ECR credentials at secret/aws-ecr"

echo "Vault initialization complete!"
echo ""
echo "Test credentials created:"
echo "- secret/docker-hub (Docker Hub credentials)"
echo "- secret/private-registry (Private registry credentials)"
echo "- secret/aws-ecr (AWS ECR credentials)"
echo ""
echo "To test the proxy, use username format:"
echo "docker;docker-hub;registry.hub.docker.com"
echo "With password: dev-root-token"