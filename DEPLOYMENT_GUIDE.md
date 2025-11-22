# Deployment Guide

## Quick Deployment Options

### Option 1: Local Development (Docker Compose)

```bash
# Start all services
docker-compose -f docker-compose.dev.yml up -d

# Initialize databases
make migrate-up
make seed-dev

# Check service health
make health-check
```

### Option 2: Kubernetes (Minikube/Kind)

```bash
# Create namespace
kubectl create namespace unity-collab

# Deploy infrastructure
kubectl apply -f infrastructure/kubernetes/

# Wait for pods to be ready
kubectl wait --for=condition=ready pod --all -n unity-collab --timeout=300s

# Port forward services
kubectl port-forward -n unity-collab svc/gateway 80:80
```

### Option 3: AWS EKS (Production)

```bash
# 1. Deploy infrastructure with Terraform
cd infrastructure/terraform
terraform init
terraform plan
terraform apply

# 2. Configure kubectl
aws eks update-kubeconfig --name unity-collab-production --region us-east-1

# 3. Deploy with Helm
helm install unity-collab ./infrastructure/helm/unity-collab \
  --namespace production \
  --values infrastructure/helm/unity-collab/values-production.yaml

# 4. Verify deployment
helm status unity-collab -n production
kubectl get pods -n production
```

## Pre-Deployment Checklist

- [ ] Update configuration files (`.env`, `values.yaml`)
- [ ] Set strong passwords for databases
- [ ] Configure domain names and TLS certificates
- [ ] Set up S3 buckets for assets and backups
- [ ] Configure monitoring and alerting
- [ ] Test backup and restore procedures
- [ ] Review security settings
- [ ] Update DNS records
- [ ] Prepare rollback plan

## Environment-Specific Deployments

### Development

```bash
helm install collab-dev ./infrastructure/helm/unity-collab \
  --namespace development \
  --values values-dev.yaml
```

### Staging

```bash
helm install collab-staging ./infrastructure/helm/unity-collab \
  --namespace staging \
  --values values-staging.yaml
```

### Production

```bash
helm install collab-prod ./infrastructure/helm/unity-collab \
  --namespace production \
  --values values-production.yaml \
  --wait --timeout 10m
```

## Monitoring Deployment

### Check Pod Status

```bash
kubectl get pods -n unity-collab -w
```

### View Logs

```bash
# All services
kubectl logs -f -l tier=backend -n unity-collab

# Specific service
kubectl logs -f -l app=auth-service -n unity-collab
```

### Check Metrics

```bash
# Access Prometheus
kubectl port-forward -n unity-collab svc/prometheus 9090:9090

# Access Grafana
kubectl port-forward -n unity-collab svc/grafana 3000:3000
```

## Database Migrations

### Run Migrations

```bash
# Manually run migrations
kubectl exec -it -n unity-collab deployment/session-service -- \
  /app/migrate -path=/migrations -database="postgresql://..." up

# Or use migration job
kubectl apply -f infrastructure/kubernetes/migration-job.yaml
```

## Backup and Restore

### Create Backup

```bash
./scripts/backup-database.sh
```

### Restore from Backup

```bash
./scripts/restore-database.sh 20250122_143000 postgres
```

## Scaling

### Manual Scaling

```bash
kubectl scale deployment auth-service --replicas=5 -n unity-collab
```

### Auto-Scaling

HPA is configured automatically. Monitor with:

```bash
kubectl get hpa -n unity-collab
```

## Rollback

### Helm Rollback

```bash
# List releases
helm history unity-collab -n production

# Rollback to previous version
helm rollback unity-collab -n production

# Rollback to specific revision
helm rollback unity-collab 3 -n production
```

### Kubernetes Rollback

```bash
kubectl rollout undo deployment/auth-service -n unity-collab
```

## Troubleshooting

### Pod Not Starting

```bash
kubectl describe pod <pod-name> -n unity-collab
kubectl logs <pod-name> -n unity-collab
```

### Service Unavailable

```bash
kubectl get svc -n unity-collab
kubectl get endpoints -n unity-collab
```

### Database Connection Issues

```bash
kubectl exec -it deployment/auth-service -n unity-collab -- \
  nc -zv postgres 5432
```

## Security

### TLS Certificates

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Create certificate
kubectl apply -f infrastructure/kubernetes/certificates.yaml
```

### Secrets Management

```bash
# Create secrets
kubectl create secret generic unity-collab-secrets \
  --from-literal=postgres-password=<password> \
  --from-literal=redis-password=<password> \
  -n unity-collab

# Or use external secrets operator
kubectl apply -f infrastructure/kubernetes/external-secrets.yaml
```

## Post-Deployment

1. **Verify all services are healthy**
2. **Run smoke tests**
3. **Check monitoring dashboards**
4. **Verify backup jobs are running**
5. **Test critical user flows**
6. **Update documentation**
7. **Notify stakeholders**

## Maintenance

### Update Services

```bash
# Update image tag
helm upgrade unity-collab ./infrastructure/helm/unity-collab \
  --set image.tag=v1.2.0 \
  --namespace production
```

### Database Maintenance

```bash
# Vacuum and analyze
kubectl exec -it deployment/postgres -n unity-collab -- \
  psql -U postgres -c "VACUUM ANALYZE;"
```

## Disaster Recovery

See [DEPLOYMENT.md](./DEPLOYMENT.md) for complete DR procedures.

Quick DR:
1. Restore latest backup
2. Verify data integrity
3. Restart services
4. Run health checks
5. Notify users of restoration
