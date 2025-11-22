# Unity Collaboration Platform - Deployment Architecture

## Table of Contents

1. [Infrastructure Overview](#infrastructure-overview)
2. [Kubernetes Setup](#kubernetes-setup)
3. [Service Deployments](#service-deployments)
4. [CI/CD Pipeline](#cicd-pipeline)
5. [Monitoring & Observability](#monitoring--observability)
6. [Disaster Recovery](#disaster-recovery)
7. [Security & Compliance](#security--compliance)
8. [Cost Optimization](#cost-optimization)

---

## Infrastructure Overview

### Multi-Region Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Global DNS (Route 53)                        │
│              Geo-routing + Health Checks                         │
└─────────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          │                   │                   │
    ┌─────▼──────┐     ┌─────▼──────┐     ┌─────▼──────┐
    │  US-East-1 │     │  EU-West-1 │     │ AP-South-1 │
    │   Region   │     │   Region   │     │   Region   │
    └────────────┘     └────────────┘     └────────────┘
          │                   │                   │
    ┌─────▼──────┐     ┌─────▼──────┐     ┌─────▼──────┐
    │ EKS Cluster│     │ EKS Cluster│     │ EKS Cluster│
    │ 3 AZs      │     │ 3 AZs      │     │ 3 AZs      │
    └────────────┘     └────────────┘     └────────────┘
          │                   │                   │
          └───────────────────┼───────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │  Global Database  │
                    │  Aurora Global    │
                    │  Multi-Master     │
                    └───────────────────┘
```

### AWS Infrastructure

**Core Services**
- **Compute**: EKS (Elastic Kubernetes Service)
- **Database**: Aurora PostgreSQL (Multi-Master)
- **Cache**: ElastiCache Redis (Cluster Mode)
- **Storage**: S3 + CloudFront CDN
- **Message Queue**: Amazon MQ (RabbitMQ)
- **Load Balancer**: Application Load Balancer
- **DNS**: Route 53
- **Monitoring**: CloudWatch + Prometheus + Grafana
- **Logging**: ELK Stack on EC2
- **Secrets**: AWS Secrets Manager

---

## Kubernetes Setup

### Cluster Configuration

**Production Cluster**
```yaml
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig

metadata:
  name: collab-prod-us-east-1
  region: us-east-1
  version: "1.28"

vpc:
  cidr: 10.0.0.0/16
  nat:
    gateway: HighlyAvailable

iam:
  withOIDC: true

managedNodeGroups:
  - name: api-services
    instanceType: t3.large
    minSize: 5
    maxSize: 20
    desiredCapacity: 10
    volumeSize: 50
    labels:
      workload: api
    tags:
      k8s.io/cluster-autoscaler/enabled: "true"

  - name: realtime-services
    instanceType: c5.xlarge
    minSize: 10
    maxSize: 50
    desiredCapacity: 20
    volumeSize: 50
    labels:
      workload: realtime
    tags:
      k8s.io/cluster-autoscaler/enabled: "true"

  - name: background-jobs
    instanceType: t3.medium
    minSize: 3
    maxSize: 10
    desiredCapacity: 5
    volumeSize: 100
    labels:
      workload: jobs

addons:
  - name: vpc-cni
  - name: coredns
  - name: kube-proxy
  - name: aws-ebs-csi-driver

cloudWatch:
  clusterLogging:
    enableTypes: ["api", "audit", "authenticator", "controllerManager", "scheduler"]
```

### Namespaces

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: production
  labels:
    environment: production
---
apiVersion: v1
kind: Namespace
metadata:
  name: staging
  labels:
    environment: staging
---
apiVersion: v1
kind: Namespace
metadata:
  name: monitoring
  labels:
    environment: production
---
apiVersion: v1
kind: Namespace
metadata:
  name: ingress-nginx
  labels:
    environment: production
```

### Resource Quotas

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: production-quota
  namespace: production
spec:
  hard:
    requests.cpu: "100"
    requests.memory: 200Gi
    limits.cpu: "200"
    limits.memory: 400Gi
    persistentvolumeclaims: "50"
    services.loadbalancers: "5"
```

---

## Service Deployments

### Session Service Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: session-service
  namespace: production
  labels:
    app: session-service
    version: v1.0.0
spec:
  replicas: 5
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 2
      maxUnavailable: 1
  selector:
    matchLabels:
      app: session-service
  template:
    metadata:
      labels:
        app: session-service
        version: v1.0.0
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
    spec:
      serviceAccountName: session-service-sa

      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchExpressions:
                    - key: app
                      operator: In
                      values:
                        - session-service
                topologyKey: kubernetes.io/hostname

      containers:
        - name: session-service
          image: 123456789.dkr.ecr.us-east-1.amazonaws.com/session-service:v1.0.0
          imagePullPolicy: Always

          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
            - name: grpc
              containerPort: 9090
              protocol: TCP
            - name: metrics
              containerPort: 9091
              protocol: TCP

          env:
            - name: ENVIRONMENT
              value: "production"
            - name: LOG_LEVEL
              value: "info"
            - name: DB_HOST
              valueFrom:
                secretKeyRef:
                  name: database-credentials
                  key: host
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: database-credentials
                  key: password
            - name: REDIS_URL
              valueFrom:
                configMapKeyRef:
                  name: redis-config
                  key: url
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: jwt-secret
                  key: secret

          resources:
            requests:
              cpu: 500m
              memory: 512Mi
            limits:
              cpu: 1000m
              memory: 1Gi

          livenessProbe:
            httpGet:
              path: /health/live
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3

          readinessProbe:
            httpGet:
              path: /health/ready
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 3

          volumeMounts:
            - name: config
              mountPath: /app/config
              readOnly: true

      volumes:
        - name: config
          configMap:
            name: session-service-config
---
apiVersion: v1
kind: Service
metadata:
  name: session-service
  namespace: production
  labels:
    app: session-service
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
spec:
  type: LoadBalancer
  selector:
    app: session-service
  ports:
    - name: http
      protocol: TCP
      port: 80
      targetPort: 8080
    - name: grpc
      protocol: TCP
      port: 9090
      targetPort: 9090
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: session-service-hpa
  namespace: production
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: session-service
  minReplicas: 5
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
        - type: Percent
          value: 50
          periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 25
          periodSeconds: 60
```

### Sync Service (WebSocket) Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sync-service
  namespace: production
spec:
  replicas: 20
  selector:
    matchLabels:
      app: sync-service
  template:
    metadata:
      labels:
        app: sync-service
    spec:
      nodeSelector:
        workload: realtime

      containers:
        - name: sync-service
          image: 123456789.dkr.ecr.us-east-1.amazonaws.com/sync-service:v1.0.0

          ports:
            - name: websocket
              containerPort: 8080
            - name: metrics
              containerPort: 9090

          env:
            - name: NODE_ENV
              value: "production"
            - name: REDIS_URL
              valueFrom:
                secretKeyRef:
                  name: redis-credentials
                  key: url

          resources:
            requests:
              cpu: 1000m
              memory: 2Gi
            limits:
              cpu: 2000m
              memory: 4Gi

          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 10

          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: sync-service
  namespace: production
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout: "3600"
spec:
  type: LoadBalancer
  sessionAffinity: ClientIP  # Sticky sessions for WebSocket
  selector:
    app: sync-service
  ports:
    - name: websocket
      protocol: TCP
      port: 80
      targetPort: 8080
```

### Database (PostgreSQL) StatefulSet

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: production
spec:
  serviceName: postgres
  replicas: 3
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
        - name: postgres
          image: postgres:15-alpine
          ports:
            - containerPort: 5432
              name: postgres
          env:
            - name: POSTGRES_DB
              value: collab_prod
            - name: POSTGRES_USER
              valueFrom:
                secretKeyRef:
                  name: postgres-credentials
                  key: username
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-credentials
                  key: password
            - name: PGDATA
              value: /var/lib/postgresql/data/pgdata

          volumeMounts:
            - name: postgres-storage
              mountPath: /var/lib/postgresql/data

          resources:
            requests:
              cpu: 2000m
              memory: 4Gi
            limits:
              cpu: 4000m
              memory: 8Gi

  volumeClaimTemplates:
    - metadata:
        name: postgres-storage
      spec:
        accessModes: ["ReadWriteOnce"]
        storageClassName: gp3
        resources:
          requests:
            storage: 500Gi
```

### Ingress Configuration

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-ingress
  namespace: production
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
spec:
  tls:
    - hosts:
        - api.collab.example.com
      secretName: api-tls
  rules:
    - host: api.collab.example.com
      http:
        paths:
          - path: /v1/sessions
            pathType: Prefix
            backend:
              service:
                name: session-service
                port:
                  number: 80
          - path: /v1/assets
            pathType: Prefix
            backend:
              service:
                name: asset-service
                port:
                  number: 80
          - path: /v1/auth
            pathType: Prefix
            backend:
              service:
                name: auth-service
                port:
                  number: 80
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: websocket-ingress
  namespace: production
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/websocket-services: sync-service
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
spec:
  tls:
    - hosts:
        - sync.collab.example.com
      secretName: sync-tls
  rules:
    - host: sync.collab.example.com
      http:
        paths:
          - path: /ws
            pathType: Prefix
            backend:
              service:
                name: sync-service
                port:
                  number: 80
```

---

## CI/CD Pipeline

### GitLab CI Configuration

```yaml
# .gitlab-ci.yml
stages:
  - test
  - build
  - deploy-staging
  - deploy-production

variables:
  DOCKER_REGISTRY: 123456789.dkr.ecr.us-east-1.amazonaws.com
  APP_NAME: session-service

# Test stage
test:unit:
  stage: test
  image: golang:1.21
  script:
    - go test -v -cover ./...
  only:
    - merge_requests
    - main

test:integration:
  stage: test
  image: golang:1.21
  services:
    - postgres:15
    - redis:7
  script:
    - go test -v -tags=integration ./tests/integration/...
  only:
    - merge_requests
    - main

# Build stage
build:
  stage: build
  image: docker:latest
  services:
    - docker:dind
  before_script:
    - aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin $DOCKER_REGISTRY
  script:
    - docker build -t $DOCKER_REGISTRY/$APP_NAME:$CI_COMMIT_SHA .
    - docker tag $DOCKER_REGISTRY/$APP_NAME:$CI_COMMIT_SHA $DOCKER_REGISTRY/$APP_NAME:latest
    - docker push $DOCKER_REGISTRY/$APP_NAME:$CI_COMMIT_SHA
    - docker push $DOCKER_REGISTRY/$APP_NAME:latest
  only:
    - main
    - tags

# Deploy to staging
deploy:staging:
  stage: deploy-staging
  image: bitnami/kubectl:latest
  before_script:
    - aws eks update-kubeconfig --name collab-staging-us-east-1 --region us-east-1
  script:
    - kubectl set image deployment/$APP_NAME $APP_NAME=$DOCKER_REGISTRY/$APP_NAME:$CI_COMMIT_SHA -n staging
    - kubectl rollout status deployment/$APP_NAME -n staging
  environment:
    name: staging
    url: https://staging-api.collab.example.com
  only:
    - main

# Deploy to production (manual approval)
deploy:production:
  stage: deploy-production
  image: bitnami/kubectl:latest
  before_script:
    - aws eks update-kubeconfig --name collab-prod-us-east-1 --region us-east-1
  script:
    - kubectl set image deployment/$APP_NAME $APP_NAME=$DOCKER_REGISTRY/$APP_NAME:$CI_COMMIT_SHA -n production
    - kubectl rollout status deployment/$APP_NAME -n production
  environment:
    name: production
    url: https://api.collab.example.com
  when: manual
  only:
    - tags
```

### Blue/Green Deployment

```yaml
# Blue deployment (current)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: session-service-blue
  namespace: production
spec:
  replicas: 5
  selector:
    matchLabels:
      app: session-service
      version: blue
  template:
    metadata:
      labels:
        app: session-service
        version: blue
    spec:
      containers:
        - name: session-service
          image: session-service:v1.0.0
---
# Green deployment (new version)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: session-service-green
  namespace: production
spec:
  replicas: 5
  selector:
    matchLabels:
      app: session-service
      version: green
  template:
    metadata:
      labels:
        app: session-service
        version: green
    spec:
      containers:
        - name: session-service
          image: session-service:v1.1.0
---
# Service (switch between blue/green)
apiVersion: v1
kind: Service
metadata:
  name: session-service
  namespace: production
spec:
  selector:
    app: session-service
    version: blue  # Change to 'green' to switch traffic
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080
```

---

## Monitoring & Observability

### Prometheus Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      evaluation_interval: 15s

    scrape_configs:
      - job_name: 'kubernetes-pods'
        kubernetes_sd_configs:
          - role: pod
        relabel_configs:
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
            action: keep
            regex: true
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
            action: replace
            target_label: __metrics_path__
            regex: (.+)
          - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
            action: replace
            regex: ([^:]+)(?::\d+)?;(\d+)
            replacement: $1:$2
            target_label: __address__

      - job_name: 'kubernetes-nodes'
        kubernetes_sd_configs:
          - role: node
        relabel_configs:
          - action: labelmap
            regex: __meta_kubernetes_node_label_(.+)

    alerting:
      alertmanagers:
        - static_configs:
            - targets: ['alertmanager:9093']

    rule_files:
      - '/etc/prometheus/rules/*.yml'
```

### Grafana Dashboards

Deploy Grafana with pre-configured dashboards:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grafana
  template:
    metadata:
      labels:
        app: grafana
    spec:
      containers:
        - name: grafana
          image: grafana/grafana:latest
          ports:
            - containerPort: 3000
          env:
            - name: GF_SECURITY_ADMIN_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: grafana-credentials
                  key: password
          volumeMounts:
            - name: grafana-storage
              mountPath: /var/lib/grafana
            - name: grafana-dashboards
              mountPath: /etc/grafana/provisioning/dashboards
      volumes:
        - name: grafana-storage
          persistentVolumeClaim:
            claimName: grafana-pvc
        - name: grafana-dashboards
          configMap:
            name: grafana-dashboards
```

---

## Disaster Recovery

### Backup Strategy

**Database Backups**
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
  namespace: production
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: backup
              image: postgres:15-alpine
              command:
                - /bin/sh
                - -c
                - |
                  BACKUP_FILE="/backup/backup_$(date +%Y%m%d_%H%M%S).sql.gz"
                  pg_dump -h $DB_HOST -U $DB_USER $DB_NAME | gzip > $BACKUP_FILE
                  aws s3 cp $BACKUP_FILE s3://collab-backups/postgres/
              env:
                - name: DB_HOST
                  value: postgres.production.svc.cluster.local
                - name: DB_USER
                  valueFrom:
                    secretKeyRef:
                      name: postgres-credentials
                      key: username
                - name: PGPASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: postgres-credentials
                      key: password
              volumeMounts:
                - name: backup-storage
                  mountPath: /backup
          volumes:
            - name: backup-storage
              emptyDir: {}
          restartPolicy: OnFailure
```

### Disaster Recovery Plan

**RTO (Recovery Time Objective)**: < 1 hour
**RPO (Recovery Point Objective)**: < 5 minutes

**Recovery Steps**:
1. Failover to secondary region
2. Restore database from latest backup
3. Replay WAL logs to minimize data loss
4. Update DNS to point to new region
5. Verify all services are operational

---

## Security & Compliance

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: session-service-policy
  namespace: production
spec:
  podSelector:
    matchLabels:
      app: session-service
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - protocol: TCP
          port: 8080
  egress:
    - to:
        - podSelector:
            matchLabels:
              app: postgres
      ports:
        - protocol: TCP
          port: 5432
    - to:
        - podSelector:
            matchLabels:
              app: redis
      ports:
        - protocol: TCP
          port: 6379
```

### Pod Security Policies

```yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: restricted
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'persistentVolumeClaim'
  hostNetwork: false
  hostIPC: false
  hostPID: false
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
  readOnlyRootFilesystem: true
```

---

## Cost Optimization

### Reserved Instances

- **EC2 instances for stable workloads**: 3-year reserved instances (50% savings)
- **RDS instances**: Reserved instances for databases
- **ElastiCache**: Reserved nodes for Redis

### Spot Instances

```yaml
managedNodeGroups:
  - name: background-jobs-spot
    instanceTypes: ["t3.medium", "t3a.medium", "t2.medium"]
    spot: true
    minSize: 3
    maxSize: 20
    labels:
      workload: jobs
      lifecycle: spot
```

### Auto-scaling Schedules

```yaml
# Scale down during low-traffic hours
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: session-service-hpa-nighttime
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: session-service
  minReplicas: 2  # Reduced from 5
  maxReplicas: 10  # Reduced from 20
```

---

This deployment architecture provides a production-ready, scalable, and resilient infrastructure for the Unity Collaboration Platform.
