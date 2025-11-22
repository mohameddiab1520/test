# Unity Collaboration Platform - Terraform Infrastructure

This directory contains Terraform configurations for deploying the Unity Collaboration Platform infrastructure on AWS.

## Architecture

The infrastructure includes:
- **VPC** with public and private subnets across 3 availability zones
- **EKS Cluster** for Kubernetes orchestration
- **RDS PostgreSQL** for relational data
- **ElastiCache Redis** for caching and real-time sessions
- **S3** for asset storage
- **CloudFront** CDN for global asset distribution
- **Secrets Manager** for secure credential storage

## Prerequisites

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [AWS CLI](https://aws.amazon.com/cli/) configured with appropriate credentials
- [kubectl](https://kubernetes.io/docs/tasks/tools/) for Kubernetes management

## Quick Start

### 1. Initialize Terraform

```bash
cd infrastructure/terraform
terraform init
```

### 2. Create Terraform Variables File

```bash
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars` with your values:

```hcl
aws_region         = "us-east-1"
environment        = "production"
project_name       = "unity-collab"
eks_min_nodes      = 3
eks_max_nodes      = 10
eks_desired_nodes  = 3
```

### 3. Plan Infrastructure

```bash
terraform plan
```

### 4. Apply Infrastructure

```bash
terraform apply
```

Type `yes` when prompted to confirm.

### 5. Configure kubectl

After the infrastructure is created, configure kubectl:

```bash
aws eks update-kubeconfig --region us-east-1 --name unity-collab-production
```

### 6. Verify Cluster Access

```bash
kubectl get nodes
```

## Modules

### VPC (`vpc.tf`)
- Creates VPC with public and private subnets
- Sets up NAT gateways for outbound internet access
- Configures route tables and security groups

### EKS (`eks.tf`)
- Creates EKS cluster with managed node groups
- Configures OIDC provider for service accounts
- Installs essential add-ons (VPC CNI, CoreDNS, kube-proxy)

### RDS (`rds.tf`)
- PostgreSQL 15 instance
- Multi-AZ deployment for high availability
- Automated backups and performance insights
- Encrypted storage

### ElastiCache (`elasticache.tf`)
- Redis 7.0 cluster
- Replication for high availability
- Encryption at rest and in transit

### S3 (`s3.tf`)
- Asset storage bucket
- CloudFront CDN distribution
- Lifecycle policies for cost optimization
- Versioning enabled

## Variables

Key variables you can customize:

| Variable | Description | Default |
|----------|-------------|---------|
| `aws_region` | AWS region | us-east-1 |
| `environment` | Environment name | production |
| `vpc_cidr` | VPC CIDR block | 10.0.0.0/16 |
| `eks_cluster_version` | Kubernetes version | 1.28 |
| `eks_node_instance_types` | EC2 instance types | ["t3.medium"] |
| `rds_instance_class` | RDS instance class | db.t3.medium |
| `elasticache_node_type` | Redis node type | cache.t3.medium |

See `variables.tf` for complete list.

## Outputs

After applying, Terraform outputs important values:

```bash
# View all outputs
terraform output

# View specific output
terraform output eks_cluster_endpoint
```

## Cost Estimation

Approximate monthly costs (us-east-1):
- EKS Control Plane: $73/month
- EKS Worker Nodes (3 x t3.medium): ~$100/month
- RDS (db.t3.medium): ~$60/month
- ElastiCache (2 x cache.t3.medium): ~$100/month
- S3 + CloudFront: Variable (depends on usage)

**Total: ~$350-500/month** (excluding S3/CloudFront usage)

## Security

- All data encrypted at rest
- TLS encryption in transit
- Secrets stored in AWS Secrets Manager
- IAM roles following least privilege principle
- VPC with private subnets for databases
- Security groups restricting access

## Backup and Disaster Recovery

- **RDS**: Automated daily backups with 7-day retention
- **ElastiCache**: Daily snapshots with 7-day retention
- **S3**: Versioning enabled
- **Multi-AZ**: RDS and ElastiCache deployed across AZs

## Monitoring

CloudWatch monitoring enabled for:
- EKS cluster and node metrics
- RDS performance insights
- VPC flow logs
- S3 access logs

## Updating Infrastructure

```bash
# Modify variables or configuration
terraform plan

# Apply changes
terraform apply

# Target specific resource
terraform apply -target=aws_eks_node_group.main
```

## Destroying Infrastructure

**Warning**: This will delete all resources!

```bash
terraform destroy
```

## State Management

Terraform state is stored remotely in S3 with DynamoDB locking:
- Bucket: `unity-collab-terraform-state`
- DynamoDB Table: `unity-collab-terraform-locks`

## Troubleshooting

### Issue: EKS nodes not joining cluster

```bash
# Check node status
kubectl get nodes

# View logs
kubectl logs -n kube-system -l k8s-app=aws-node
```

### Issue: RDS connection timeout

- Verify security group allows traffic from EKS cluster
- Check VPC route tables
- Verify RDS is in private subnet

### Issue: S3 bucket access denied

- Check IAM roles and policies
- Verify CloudFront OAI has correct permissions

## Additional Resources

- [AWS EKS Documentation](https://docs.aws.amazon.com/eks/)
- [Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
- [Kubernetes Documentation](https://kubernetes.io/docs/home/)

## Support

For issues or questions:
- Open an issue in the repository
- Contact: infrastructure@collab.example.com
