#!/bin/bash

echo "🧹 Cleaning up Vault Docker Proxy deployment..."

# Delete all resources
kubectl delete -f . -n vault-docker-proxy --ignore-not-found=true

# Delete namespace
kubectl delete namespace vault-docker-proxy --ignore-not-found=true

echo "✅ Cleanup complete!"