# RDS Subnet Group
resource "aws_db_subnet_group" "main" {
  name       = "${var.project_name}-db-subnet-${var.environment}"
  subnet_ids = aws_subnet.private[*].id

  tags = merge(
    var.tags,
    {
      Name = "${var.project_name}-db-subnet-${var.environment}"
    }
  )
}

# RDS Security Group
resource "aws_security_group" "rds" {
  name        = "${var.project_name}-rds-sg-${var.environment}"
  description = "Security group for RDS PostgreSQL"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.eks_cluster.id]
    description     = "PostgreSQL from EKS"
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
      Name = "${var.project_name}-rds-sg-${var.environment}"
    }
  )
}

# RDS Parameter Group
resource "aws_db_parameter_group" "main" {
  name   = "${var.project_name}-postgres15-${var.environment}"
  family = "postgres15"

  parameter {
    name  = "log_connections"
    value = "1"
  }

  parameter {
    name  = "log_disconnections"
    value = "1"
  }

  parameter {
    name  = "log_statement"
    value = "all"
  }

  parameter {
    name  = "max_connections"
    value = "200"
  }

  parameter {
    name  = "shared_buffers"
    value = "{DBInstanceClassMemory/10922}"
  }

  tags = var.tags
}

# RDS Instance
resource "aws_db_instance" "main" {
  identifier     = "${var.project_name}-postgres-${var.environment}"
  engine         = "postgres"
  engine_version = "15.4"
  instance_class = var.rds_instance_class

  allocated_storage     = var.rds_allocated_storage
  max_allocated_storage = var.rds_allocated_storage * 2
  storage_type          = "gp3"
  storage_encrypted     = true

  db_name  = "collab_prod"
  username = "collab_admin"
  password = random_password.rds_password.result

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  parameter_group_name   = aws_db_parameter_group.main.name

  backup_retention_period = var.backup_retention_days
  backup_window          = "03:00-04:00"
  maintenance_window     = "sun:04:00-sun:05:00"

  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]
  performance_insights_enabled    = true
  monitoring_interval             = var.enable_monitoring ? 60 : 0
  monitoring_role_arn            = var.enable_monitoring ? aws_iam_role.rds_monitoring[0].arn : null

  multi_az               = var.environment == "production" ? true : false
  deletion_protection    = var.environment == "production" ? true : false
  skip_final_snapshot    = var.environment != "production"
  final_snapshot_identifier = var.environment == "production" ? "${var.project_name}-final-snapshot-${formatdate("YYYY-MM-DD-hhmm", timestamp())}" : null

  tags = merge(
    var.tags,
    {
      Name = "${var.project_name}-postgres-${var.environment}"
    }
  )
}

# Random Password for RDS
resource "random_password" "rds_password" {
  length  = 32
  special = true
}

# Store RDS password in Secrets Manager
resource "aws_secretsmanager_secret" "rds_password" {
  name = "${var.project_name}/rds/password-${var.environment}"

  tags = var.tags
}

resource "aws_secretsmanager_secret_version" "rds_password" {
  secret_id = aws_secretsmanager_secret.rds_password.id
  secret_string = jsonencode({
    username = aws_db_instance.main.username
    password = random_password.rds_password.result
    host     = aws_db_instance.main.address
    port     = aws_db_instance.main.port
    dbname   = aws_db_instance.main.db_name
  })
}

# RDS Monitoring Role
resource "aws_iam_role" "rds_monitoring" {
  count = var.enable_monitoring ? 1 : 0

  name = "${var.project_name}-rds-monitoring-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "monitoring.rds.amazonaws.com"
      }
    }]
  })

  tags = var.tags
}

resource "aws_iam_role_policy_attachment" "rds_monitoring" {
  count = var.enable_monitoring ? 1 : 0

  role       = aws_iam_role.rds_monitoring[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonRDSEnhancedMonitoringRole"
}
