#!/bin/bash

set -e

echo "🚀 Deploying Vault Docker Proxy with Dev Vault in Kubernetes..."

# Create namespace if it doesn't exist
echo "📝 Creating namespace..."
kubectl create namespace vault-docker-proxy --dry-run=client -o yaml | kubectl apply -f -

# Deploy Vault dev server first
echo "🔐 Deploying Vault dev server..."
kubectl apply -f vault-dev.yaml -n vault-docker-proxy

# Wait for Vault to be ready
echo "⏳ Waiting for Vault to be ready..."
kubectl wait --for=condition=ready pod -l app=vault-dev -n vault-docker-proxy --timeout=60s

# Deploy the proxy configuration and deployment
echo "🔧 Deploying proxy configuration..."
kubectl apply -f configmap.yaml -n vault-docker-proxy

echo "🚀 Deploying vault-docker-proxy..."
kubectl apply -f deployment.yaml -n vault-docker-proxy
kubectl apply -f service.yaml -n vault-docker-proxy

# Initialize Vault with test credentials
echo "📋 Initializing Vault with test credentials..."
kubectl apply -f vault-init-job.yaml -n vault-docker-proxy

# Wait for init job to complete
echo "⏳ Waiting for Vault initialization to complete..."
kubectl wait --for=condition=complete job/vault-init -n vault-docker-proxy --timeout=120s

# Show the init job logs
echo "📄 Vault initialization logs:"
kubectl logs job/vault-init -n vault-docker-proxy

# Wait for proxy to be ready
echo "⏳ Waiting for proxy to be ready..."
kubectl wait --for=condition=ready pod -l app=vault-docker-proxy -n vault-docker-proxy --timeout=60s

echo ""
echo "✅ Deployment complete!"
echo ""
echo "📊 Deployment status:"
kubectl get pods,svc -n vault-docker-proxy

echo ""
echo "🔍 Testing the deployment:"
echo ""
echo "1. Get the service IP:"
echo "   kubectl get svc vault-docker-proxy-lb -n vault-docker-proxy"
echo ""
echo "2. Test the API:"
echo "   curl -u \"docker;docker-hub;registry.hub.docker.com:dev-root-token\" \\"
echo "     http://<SERVICE-IP>/v2/"
echo ""
echo "3. Test tag listing:"
echo "   curl -u \"docker;docker-hub;registry.hub.docker.com:dev-root-token\" \\"
echo "     \"http://<SERVICE-IP>/v2/library/nginx/tags/list\""
echo ""
echo "📝 Available test credentials in Vault:"
echo "   - secret/docker-hub (username: your-docker-username)"
echo "   - secret/private-registry (username: registry-user)"
echo "   - secret/aws-ecr (username: AWS)"
echo "   - secret/custom-registry (username: custom-user)"
echo ""
echo "🔑 Vault token for testing: dev-root-token"
echo "🌐 Vault UI (if port-forwarded): http://localhost:8200"
echo ""
echo "To port-forward Vault:"
echo "   kubectl port-forward svc/vault-dev 8200:8200 -n vault-docker-proxy"
echo ""
echo "To view logs:"
echo "   kubectl logs -f deployment/vault-docker-proxy -n vault-docker-proxy"