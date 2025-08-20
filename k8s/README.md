# Kubernetes Deployment for Vault Docker Registry Proxy

This directory contains Kubernetes manifests to deploy the vault-docker-proxy in your Kubernetes cluster.

## Files

- `deployment.yaml` - Main deployment with 2 replicas, health checks, and security context
- `service.yaml` - ClusterIP and LoadBalancer services
- `configmap.yaml` - Configuration for Vault server address
- `ingress.yaml` - Optional ingress for external access
- `kustomization.yaml` - Kustomize configuration for easy deployment

## Prerequisites

1. **Docker Image**: Build and push the Docker image to your registry:
   ```bash
   # From the project root
   docker build -f docker/Dockerfile -t your-registry/vault-docker-proxy:latest .
   docker push your-registry/vault-docker-proxy:latest
   ```

2. **Vault Server**: Either use the included dev server or have a Vault server running in your cluster.

## Quick Development Setup (with Vault Dev Server)

For testing and development, you can deploy everything including a Vault dev server:

```bash
# Build and load the image (for local testing)
docker build -f docker/Dockerfile -t vault-docker-proxy:latest .

# Option 1: Use the automated deployment script
./deploy-dev.sh

# Option 2: Deploy manually
kubectl apply -k . --create-namespace
```

This will deploy:
- Vault dev server with root token `dev-root-token`
- Vault initialization job with test credentials
- vault-docker-proxy configured to use the dev Vault
- Services for both Vault and proxy

### Test Credentials Available

The dev setup includes these test credentials in Vault:
- `secret/docker-hub` - Docker Hub credentials
- `secret/private-registry` - Private registry credentials  
- `secret/aws-ecr` - AWS ECR credentials
- `secret/custom-registry` - Custom registry credentials

## Quick Deployment

### Option 1: Using kubectl

1. Create namespace:
   ```bash
   kubectl create namespace vault-docker-proxy
   ```

2. Update the ConfigMap with your Vault server address:
   ```bash
   kubectl edit configmap vault-docker-proxy-config -n vault-docker-proxy
   ```

3. Deploy all resources:
   ```bash
   kubectl apply -f . -n vault-docker-proxy
   ```

### Option 2: Using Kustomize

1. Update `kustomization.yaml` with your image registry:
   ```yaml
   images:
   - name: vault-docker-proxy
     newName: your-registry/vault-docker-proxy
     newTag: latest
   ```

2. Deploy:
   ```bash
   kubectl apply -k . --create-namespace
   ```

## Configuration

### Vault Server Address

Update the `vault-addr` in `configmap.yaml` to point to your Vault server:

```yaml
data:
  vault-addr: "https://your-vault-server:8200"
```

### Scaling

Adjust replicas in `deployment.yaml`:

```yaml
spec:
  replicas: 3  # Increase for higher availability
```

### Resource Limits

Modify resource requests/limits in `deployment.yaml` based on your needs:

```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "200m"
  limits:
    memory: "256Mi"
    cpu: "500m"
```

## Access Methods

### 1. ClusterIP Service (Internal)
- Access from within cluster: `http://vault-docker-proxy.vault-docker-proxy.svc.cluster.local:8080`

### 2. LoadBalancer Service (External)
- Get external IP: `kubectl get svc vault-docker-proxy-lb -n vault-docker-proxy`
- Access: `http://<EXTERNAL-IP>/v2/`

### 3. Ingress (Custom Domain)
- Update `ingress.yaml` with your domain
- Configure TLS if needed
- Access: `https://registry.yourdomain.com/v2/`

## Health Checks

The deployment includes:
- **Liveness Probe**: Checks `/v2/` endpoint every 30s
- **Readiness Probe**: Checks `/v2/` endpoint every 10s

## Security Features

- Runs as non-root user (65534)
- Read-only root filesystem
- Drops all Linux capabilities
- No privilege escalation

## Testing

Test the deployment:

```bash
# Get service endpoint
kubectl get svc -n vault-docker-proxy

# Test API version endpoint
curl -u "docker;docker-hub;registry.hub.docker.com:your-vault-token" \
  http://<SERVICE-IP>:8080/v2/
```

## Monitoring

View logs:
```bash
kubectl logs -f deployment/vault-docker-proxy -n vault-docker-proxy
```

Check pod status:
```bash
kubectl get pods -n vault-docker-proxy
```

## Troubleshooting

### Common Issues

1. **Image Pull Errors**: Ensure the Docker image is built and pushed to accessible registry
2. **Vault Connection Issues**: Check the `vault-addr` in ConfigMap
3. **Service Discovery**: Verify service names and namespaces match your Vault setup

### Debug Commands

```bash
# Check deployment status
kubectl describe deployment vault-docker-proxy -n vault-docker-proxy

# Check service endpoints
kubectl get endpoints vault-docker-proxy -n vault-docker-proxy

# View pod environment variables
kubectl exec deployment/vault-docker-proxy -n vault-docker-proxy -- env
```