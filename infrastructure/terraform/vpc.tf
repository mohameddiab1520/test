# VPC
resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = merge(
    var.tags,
    {
      Name = "${var.project_name}-vpc-${var.environment}"
    }
  )
}

# Internet Gateway
resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = merge(
    var.tags,
    {
      Name = "${var.project_name}-igw-${var.environment}"
    }
  )
}

# Public Subnets
resource "aws_subnet" "public" {
  count = var.availability_zones_count

  vpc_id                  = aws_vpc.main.id
  cidr_block              = cidrsubnet(var.vpc_cidr, 4, count.index)
  availability_zone       = data.aws_availability_zones.available.names[count.index]
  map_public_ip_on_launch = true

  tags = merge(
    var.tags,
    {
      Name                                           = "${var.project_name}-public-${count.index + 1}-${var.environment}"
      "kubernetes.io/role/elb"                       = "1"
      "kubernetes.io/cluster/${var.project_name}"    = "shared"
    }
  )
}

# Private Subnets
resource "aws_subnet" "private" {
  count = var.availability_zones_count

  vpc_id            = aws_vpc.main.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 4, count.index + var.availability_zones_count)
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = merge(
    var.tags,
    {
      Name                                           = "${var.project_name}-private-${count.index + 1}-${var.environment}"
      "kubernetes.io/role/internal-elb"              = "1"
      "kubernetes.io/cluster/${var.project_name}"    = "shared"
    }
  )
}

# Elastic IPs for NAT Gateways
resource "aws_eip" "nat" {
  count  = var.availability_zones_count
  domain = "vpc"

  tags = merge(
    var.tags,
    {
      Name = "${var.project_name}-nat-eip-${count.index + 1}-${var.environment}"
    }
  )

  depends_on = [aws_internet_gateway.main]
}

# NAT Gateways
resource "aws_nat_gateway" "main" {
  count = var.availability_zones_count

  allocation_id = aws_eip.nat[count.index].id
  subnet_id     = aws_subnet.public[count.index].id

  tags = merge(
    var.tags,
    {
      Name = "${var.project_name}-nat-${count.index + 1}-${var.environment}"
    }
  )

  depends_on = [aws_internet_gateway.main]
}

# Public Route Table
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  tags = merge(
    var.tags,
    {
      Name = "${var.project_name}-public-rt-${var.environment}"
    }
  )
}

# Public Route
resource "aws_route" "public" {
  route_table_id         = aws_route_table.public.id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.main.id
}

# Public Route Table Associations
resource "aws_route_table_association" "public" {
  count = var.availability_zones_count

  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

# Private Route Tables
resource "aws_route_table" "private" {
  count = var.availability_zones_count

  vpc_id = aws_vpc.main.id

  tags = merge(
    var.tags,
    {
      Name = "${var.project_name}-private-rt-${count.index + 1}-${var.environment}"
    }
  )
}

# Private Routes
resource "aws_route" "private" {
  count = var.availability_zones_count

  route_table_id         = aws_route_table.private[count.index].id
  destination_cidr_block = "0.0.0.0/0"
  nat_gateway_id         = aws_nat_gateway.main[count.index].id
}

# Private Route Table Associations
resource "aws_route_table_association" "private" {
  count = var.availability_zones_count

  subnet_id      = aws_subnet.private[count.index].id
  route_table_id = aws_route_table.private[count.index].id
}

# VPC Flow Logs
resource "aws_flow_log" "main" {
  count = var.enable_monitoring ? 1 : 0

  iam_role_arn    = aws_iam_role.vpc_flow_logs[0].arn
  log_destination = aws_cloudwatch_log_group.vpc_flow_logs[0].arn
  traffic_type    = "ALL"
  vpc_id          = aws_vpc.main.id

  tags = merge(
    var.tags,
    {
      Name = "${var.project_name}-vpc-flow-logs-${var.environment}"
    }
  )
}

resource "aws_cloudwatch_log_group" "vpc_flow_logs" {
  count = var.enable_monitoring ? 1 : 0

  name              = "/aws/vpc/${var.project_name}-${var.environment}"
  retention_in_days = 7

  tags = var.tags
}

resource "aws_iam_role" "vpc_flow_logs" {
  count = var.enable_monitoring ? 1 : 0

  name = "${var.project_name}-vpc-flow-logs-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "vpc-flow-logs.amazonaws.com"
        }
      }
    ]
  })

  tags = var.tags
}

resource "aws_iam_role_policy" "vpc_flow_logs" {
  count = var.enable_monitoring ? 1 : 0

  name = "${var.project_name}-vpc-flow-logs-${var.environment}"
  role = aws_iam_role.vpc_flow_logs[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogGroups",
          "logs:DescribeLogStreams"
        ]
        Effect   = "Allow"
        Resource = "*"
      }
    ]
  })
}
