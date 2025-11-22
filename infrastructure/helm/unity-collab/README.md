# Unity Collaboration Platform Helm Chart

This Helm chart deploys the complete Unity Collaboration Platform on Kubernetes.

## Prerequisites

- Kubernetes 1.23+
- Helm 3.8+
- PV provisioner support in the underlying infrastructure
- Ingress controller (nginx recommended)
- Cert-manager (optional, for TLS certificates)

## Installing the Chart

### Quick Start

```bash
# Add the Helm repository (if published)
helm repo add unity-collab https://charts.collab.example.com
helm repo update

# Install the chart
helm install my-collab unity-collab/unity-collab
```

### From Source

```bash
# Navigate to helm directory
cd infrastructure/helm

# Install the chart
helm install my-collab ./unity-collab

# Install with custom values
helm install my-collab ./unity-collab -f custom-values.yaml

# Install in specific namespace
helm install my-collab ./unity-collab --namespace unity-collab --create-namespace
```

## Configuration

### Important Configuration Values

Create a `custom-values.yaml` file:

```yaml
global:
  environment: production
  domain: your-domain.com

# Database passwords (use strong passwords in production!)
secrets:
  postgresql:
    password: "your-strong-password"
  redis:
    password: "your-strong-password"
  jwt:
    secret: "your-jwt-secret"
    refreshSecret: "your-refresh-secret"
  s3:
    accessKey: "your-s3-access-key"
    secretKey: "your-s3-secret-key"

# Ingress configuration
ingress:
  enabled: true
  hosts:
    - host: api.your-domain.com
      paths:
        - path: /
          pathType: Prefix
          service: gateway
    - host: sync.your-domain.com
      paths:
        - path: /
          pathType: Prefix
          service: sync-service

# S3 configuration
config:
  s3:
    bucket: "your-assets-bucket"
    region: "us-east-1"
```

Then install:

```bash
helm install my-collab ./unity-collab -f custom-values.yaml
```

### Complete Values Reference

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.environment` | Environment name | `production` |
| `global.domain` | Base domain | `collab.example.com` |
| `authService.enabled` | Enable auth service | `true` |
| `authService.replicaCount` | Number of replicas | `3` |
| `sessionService.enabled` | Enable session service | `true` |
| `sessionService.replicaCount` | Number of replicas | `3` |
| `assetService.enabled` | Enable asset service | `true` |
| `assetService.replicaCount` | Number of replicas | `3` |
| `syncService.enabled` | Enable sync service | `true` |
| `syncService.replicaCount` | Number of replicas | `5` |
| `gateway.enabled` | Enable gateway | `true` |
| `gateway.replicaCount` | Number of replicas | `3` |
| `postgresql.enabled` | Deploy PostgreSQL | `true` |
| `postgresql.persistence.size` | PostgreSQL storage | `50Gi` |
| `redis.enabled` | Deploy Redis | `true` |
| `redis.persistence.size` | Redis storage | `10Gi` |
| `ingress.enabled` | Enable ingress | `true` |
| `monitoring.enabled` | Enable monitoring | `true` |

See `values.yaml` for complete list.

## Upgrading

```bash
# Upgrade with new values
helm upgrade my-collab ./unity-collab -f custom-values.yaml

# Upgrade and wait for completion
helm upgrade my-collab ./unity-collab --wait

# Force recreation of resources
helm upgrade my-collab ./unity-collab --force
```

## Uninstalling

```bash
# Uninstall the release
helm uninstall my-collab

# Uninstall and delete namespace
helm uninstall my-collab --namespace unity-collab
kubectl delete namespace unity-collab
```

## Production Deployment Checklist

- [ ] Set strong passwords for all services
- [ ] Configure proper domain names
- [ ] Set up TLS certificates (cert-manager recommended)
- [ ] Configure S3 bucket and credentials
- [ ] Set up database backups
- [ ] Configure resource limits appropriately
- [ ] Enable monitoring and logging
- [ ] Review and adjust autoscaling settings
- [ ] Set up external secret management (AWS Secrets Manager, Vault)
- [ ] Configure network policies
- [ ] Set up disaster recovery procedures

## Using External Databases

To use external managed databases instead of deploying PostgreSQL/Redis:

```yaml
postgresql:
  enabled: false

# Update configmap with external endpoints
config:
  postgresql:
    host: "your-rds-endpoint.amazonaws.com"
    port: 5432
    database: "collab_prod"

redis:
  enabled: false

# Update configmap with external endpoints
config:
  redis:
    host: "your-elasticache-endpoint.amazonaws.com"
    port: 6379
```

## Monitoring

The chart includes optional Prometheus and Grafana integration:

```yaml
monitoring:
  enabled: true
  prometheus:
    enabled: true
  grafana:
    enabled: true
```

Access Grafana:

```bash
# Get Grafana password
kubectl get secret my-collab-grafana -o jsonpath="{.data.admin-password}" | base64 --decode

# Port forward to access
kubectl port-forward svc/my-collab-grafana 3000:80
```

## Troubleshooting

### Pods not starting

```bash
# Check pod status
kubectl get pods -n unity-collab

# Describe pod
kubectl describe pod <pod-name> -n unity-collab

# View logs
kubectl logs <pod-name> -n unity-collab
```

### Database connection issues

```bash
# Test PostgreSQL connection
kubectl run -it --rm debug --image=postgres:15 --restart=Never -- \
  psql -h my-collab-postgresql -U collab_user -d collab_prod

# Test Redis connection
kubectl run -it --rm debug --image=redis:7 --restart=Never -- \
  redis-cli -h my-collab-redis
```

### Ingress not working

```bash
# Check ingress
kubectl get ingress -n unity-collab
kubectl describe ingress my-collab -n unity-collab

# Check ingress controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx
```

## Support

For issues or questions:
- GitHub Issues: https://github.com/yourorg/unity-collab-platform/issues
- Email: support@collab.example.com
- Discord: https://discord.gg/example
