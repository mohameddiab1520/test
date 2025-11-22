# ElastiCache Subnet Group
resource "aws_elasticache_subnet_group" "main" {
  name       = "${var.project_name}-redis-subnet-${var.environment}"
  subnet_ids = aws_subnet.private[*].id

  tags = merge(
    var.tags,
    {
      Name = "${var.project_name}-redis-subnet-${var.environment}"
    }
  )
}

# ElastiCache Security Group
resource "aws_security_group" "elasticache" {
  name        = "${var.project_name}-redis-sg-${var.environment}"
  description = "Security group for ElastiCache Redis"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [aws_security_group.eks_cluster.id]
    description     = "Redis from EKS"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(
    var.tags,
    {
      Name = "${var.project_name}-redis-sg-${var.environment}"
    }
  )
}

# ElastiCache Parameter Group
resource "aws_elasticache_parameter_group" "main" {
  name   = "${var.project_name}-redis7-${var.environment}"
  family = "redis7"

  parameter {
    name  = "maxmemory-policy"
    value = "allkeys-lru"
  }

  parameter {
    name  = "timeout"
    value = "300"
  }

  tags = var.tags
}

# ElastiCache Replication Group
resource "aws_elasticache_replication_group" "main" {
  replication_group_id       = "${var.project_name}-redis-${var.environment}"
  replication_group_description = "Redis cluster for Unity Collaboration Platform"

  engine               = "redis"
  engine_version       = "7.0"
  node_type            = var.elasticache_node_type
  num_cache_clusters   = var.elasticache_num_nodes
  parameter_group_name = aws_elasticache_parameter_group.main.name
  port                 = 6379

  subnet_group_name    = aws_elasticache_subnet_group.main.name
  security_group_ids   = [aws_security_group.elasticache.id]

  automatic_failover_enabled = var.elasticache_num_nodes > 1
  multi_az_enabled          = var.elasticache_num_nodes > 1 && var.environment == "production"

  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  auth_token                 = random_password.redis_password.result

  snapshot_retention_limit = var.backup_retention_days
  snapshot_window         = "03:00-05:00"
  maintenance_window      = "sun:05:00-sun:07:00"

  auto_minor_version_upgrade = true
  apply_immediately         = false

  tags = merge(
    var.tags,
    {
      Name = "${var.project_name}-redis-${var.environment}"
    }
  )
}

# Random Password for Redis
resource "random_password" "redis_password" {
  length  = 32
  special = false  # Redis auth tokens don't support all special characters
}

# Store Redis password in Secrets Manager
resource "aws_secretsmanager_secret" "redis_password" {
  name = "${var.project_name}/redis/password-${var.environment}"

  tags = var.tags
}

resource "aws_secretsmanager_secret_version" "redis_password" {
  secret_id = aws_secretsmanager_secret.redis_password.id
  secret_string = jsonencode({
    password         = random_password.redis_password.result
    primary_endpoint = aws_elasticache_replication_group.main.primary_endpoint_address
    port            = aws_elasticache_replication_group.main.port
  })
}
