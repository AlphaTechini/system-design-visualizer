package terraform

import (
	"bytes"
	"fmt"
	"text/template"
)

// Generator creates Terraform configurations from architecture specs
type Generator struct {
	templates map[string]*template.Template
}

// ArchitectureSpec describes infrastructure to generate
type ArchitectureSpec struct {
	Provider    string   `json:"provider"` // aws, gcp, azure
	Region      string   `json:"region"`
	ProjectName string   `json:"project_name"`
	
	// Compute
	InstanceType    string `json:"instance_type"`
	InstanceCount   int    `json:"instance_count"`
	AMIID           string `json:"ami_id"` // AWS-specific
	
	// Database
	DatabaseEngine    string `json:"database_engine"`
	DatabaseSize      string `json:"database_size"`
	DatabaseStorageGB int    `json:"database_storage_gb"`
	MultiAZ           bool   `json:"multi_az"`
	
	// Cache
	CacheEngine  string `json:"cache_engine"` // redis, memcached
	CacheNodeType string `json:"cache_node_type"`
	CacheNodes   int    `json:"cache_nodes"`
	
	// Networking
	VPCID         string   `json:"vpc_id"`
	SubnetIDs     []string `json:"subnet_ids"`
	SecurityGroupIDs []string `json:"security_group_ids"`
	
	// Load Balancer
	LoadBalancerType string `json:"load_balancer_type"` // application, network
	HealthCheckPath  string `json:"health_check_path"`
	
	// Storage
	BucketName string `json:"bucket_name"`
	
	// Monitoring
	EnableMonitoring bool     `json:"enable_monitoring"`
	AlertEmail       string   `json:"alert_email"`
}

// NewGenerator creates Terraform generator with templates
func NewGenerator() *Generator {
	g := &Generator{
		templates: make(map[string]*template.Template),
	}
	
	// Register templates
	g.registerAWSTemplates()
	g.registerGCPTemplates()
	g.registerAzureTemplates()
	
	return g
}

// Generate creates complete Terraform configuration
func (g *Generator) Generate(spec ArchitectureSpec) (map[string]string, error) {
	files := make(map[string]string)
	
	switch spec.Provider {
	case "aws":
		return g.generateAWS(spec)
	case "gcp":
		return g.generateGCP(spec)
	case "azure":
		return g.generateAzure(spec)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", spec.Provider)
	}
	
	return files, nil
}

// generateAWS creates AWS Terraform configuration
func (g *Generator) generateAWS(spec ArchitectureSpec) (map[string]string, error) {
	files := make(map[string]string)
	
	// Generate main.tf
	mainTmpl, ok := g.templates["aws_main"]
	if !ok {
		return nil, fmt.Errorf("template aws_main not found")
	}
	
	var mainBuf bytes.Buffer
	if err := mainTmpl.Execute(&mainBuf, spec); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	files["main.tf"] = mainBuf.String()
	
	// Generate variables.tf
	variablesTmpl, ok := g.templates["aws_variables"]
	if !ok {
		return nil, fmt.Errorf("template aws_variables not found")
	}
	
	var variablesBuf bytes.Buffer
	if err := variablesTmpl.Execute(&variablesBuf, spec); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	files["variables.tf"] = variablesBuf.String()
	
	// Generate outputs.tf
	outputsTmpl, ok := g.templates["aws_outputs"]
	if !ok {
		return nil, fmt.Errorf("template aws_outputs not found")
	}
	
	var outputsBuf bytes.Buffer
	if err := outputsTmpl.Execute(&outputsBuf, spec); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	files["outputs.tf"] = outputsBuf.String()
	
	// Generate providers.tf
	providersBuf := new(bytes.Buffer)
	providersBuf.WriteString(`terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
  
  default_tags {
    tags = {
      Project     = var.project_name
      Environment = "production"
      ManagedBy   = "terraform"
    }
  }
}
`)
	files["providers.tf"] = providersBuf.String()
	
	return files, nil
}

// registerAWSTemplates registers AWS-specific templates
func (g *Generator) registerAWSTemplates() {
	// Main resources template
	g.templates["aws_main"] = template.Must(template.New("aws_main").Parse(`
# VPC and Networking
resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "${var.project_name}-vpc"
  }
}

resource "aws_subnet" "public" {
  count             = 2
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.${count.index + 1}.0/24"
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = {
    Name = "${var.project_name}-public-${count.index + 1}"
  }
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "${var.project_name}-igw"
  }
}

# Security Groups
resource "aws_security_group" "app" {
  name        = "${var.project_name}-app-sg"
  description = "Security group for application servers"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-app-sg"
  }
}

resource "aws_security_group" "db" {
  name        = "${var.project_name}-db-sg"
  description = "Security group for database"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.app.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-db-sg"
  }
}

# EC2 Auto Scaling Group
resource "aws_launch_template" "app" {
  name_prefix   = "${var.project_name}-"
  image_id      = var.ami_id
  instance_type = var.instance_type

  network_interfaces {
    associate_public_ip_address = true
    security_groups             = [aws_security_group.app.id]
  }

  user_data = base64encode(<<-EOF
              #!/bin/bash
              yum update -y
              yum install -y httpd
              systemctl start httpd
              systemctl enable httpd
              echo "<h1>${var.project_name}</h1>" > /var/www/html/index.html
              EOF
  )

  tags = {
    Name = "${var.project_name}-launch-template"
  }
}

resource "aws_autoscaling_group" "app" {
  name                = "${var.project_name}-asg"
  vpc_zone_identifier = aws_subnet.public[*].id
  target_group_arns   = [aws_lb_target_group.app.arn]
  health_check_type   = "ELB"

  min_size         = 1
  max_size         = {{.InstanceCount}}
  desired_capacity = {{.InstanceCount}}

  launch_template {
    id      = aws_launch_template.app.id
    version = "$Latest"
  }

  tag {
    key                 = "Name"
    value               = "${var.project_name}-instance"
    propagate_at_launch = true
  }
}

# Application Load Balancer
resource "aws_lb" "app" {
  name               = "${var.project_name}-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.app.id]
  subnets            = aws_subnet.public[*].id

  enable_deletion_protection = false

  tags = {
    Name = "${var.project_name}-alb"
  }
}

resource "aws_lb_target_group" "app" {
  name     = "${var.project_name}-tg"
  port     = 80
  protocol = "HTTP"
  vpc_id   = aws_vpc.main.id

  health_check {
    enabled             = true
    healthy_threshold   = 2
    interval            = 30
    matcher             = "200"
    path                = "{{.HealthCheckPath}}"
    port                = "traffic-port"
    protocol            = "HTTP"
    timeout             = 5
    unhealthy_threshold = 2
  }
}

resource "aws_lb_listener" "app" {
  load_balancer_arn = aws_lb.app.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.app.arn
  }
}

# RDS Database
resource "aws_db_subnet_group" "main" {
  name       = "${var.project_name}-db-subnet-group"
  subnet_ids = aws_subnet.public[*].id

  tags = {
    Name = "${var.project_name}-db-subnet-group"
  }
}

resource "aws_db_instance" "main" {
  identifier              = "${var.project_name}-db"
  allocated_storage       = {{.DatabaseStorageGB}}
  engine                  = "{{.DatabaseEngine}}"
  engine_version          = "15.4"
  instance_class          = "{{.DatabaseSize}}"
  db_name                 = "appdb"
  username                = "admin"
  password                = var.db_password
  parameter_group_name    = "default.postgres15"
  skip_final_snapshot     = true
  multi_az                = {{.MultiAZ}}
  db_subnet_group_name    = aws_db_subnet_group.main.name
  vpc_security_group_ids  = [aws_security_group.db.id]
  storage_encrypted       = true
  backup_retention_period = 7

  tags = {
    Name = "${var.project_name}-db"
  }
}

# ElastiCache (Redis)
{{if .CacheNodes}}
resource "aws_elasticache_cluster" "redis" {
  cluster_id           = "${var.project_name}-redis"
  engine               = "{{.CacheEngine}}"
  node_type            = "{{.CacheNodeType}}"
  num_cache_nodes      = {{.CacheNodes}}
  parameter_group_name = "default.redis7"
  port                 = 6379
  security_group_ids   = [aws_security_group.app.id]
  subnet_group_name    = aws_elasticache_subnet_group.main.name

  tags = {
    Name = "${var.project_name}-redis"
  }
}

resource "aws_elasticache_subnet_group" "main" {
  name       = "${var.project_name}-redis-subnet-group"
  subnet_ids = aws_subnet.public[*].id

  tags = {
    Name = "${var.project_name}-redis-subnet-group"
  }
}
{{end}}

# S3 Bucket
resource "aws_s3_bucket" "main" {
  bucket = "${var.project_name}-bucket"

  tags = {
    Name = "${var.project_name}-bucket"
  }
}

resource "aws_s3_bucket_versioning" "main" {
  bucket = aws_s3_bucket.main.id
  versioning_configuration {
    status = "Enabled"
  }
}

# CloudWatch Alarms
{{if .EnableMonitoring}}
resource "aws_cloudwatch_metric_alarm" "cpu_high" {
  alarm_name          = "${var.project_name}-cpu-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/EC2"
  period              = "300"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "This alarm monitors EC2 CPU utilization"
  alarm_actions       = [aws_sns_topic.alerts.arn]

  dimensions = {
    AutoScalingGroupName = aws_autoscaling_group.app.name
  }
}

resource "aws_sns_topic" "alerts" {
  name = "${var.project_name}-alerts"
}

resource "aws_sns_topic_subscription" "email" {
  topic_arn = aws_sns_topic.alerts.arn
  protocol  = "email"
  endpoint  = "{{.AlertEmail}}"
}
{{end}}

# Data Sources
data "aws_availability_zones" "available" {
  state = "available"
}
`))

	// Variables template
	g.templates["aws_variables"] = template.Must(template.New("aws_variables").Parse(`
variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Project name for resource naming"
  type        = string
}

variable "ami_id" {
  description = "AMI ID for EC2 instances"
  type        = string
  default     = "ami-0c55b159cbfafe1f0" # Amazon Linux 2
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}

variable "db_password" {
  description = "Database master password"
  type        = string
  sensitive   = true
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}
`))

	// Outputs template
	g.templates["aws_outputs"] = template.Must(template.New("aws_outputs").Parse(`
output "vpc_id" {
  description = "VPC ID"
  value       = aws_vpc.main.id
}

output "alb_dns_name" {
  description = "Application Load Balancer DNS name"
  value       = aws_lb.app.dns_name
}

output "rds_endpoint" {
  description = "RDS database endpoint"
  value       = aws_db_instance.main.endpoint
}

{{if .CacheNodes}}
output "elasticache_endpoint" {
  description = "ElastiCache Redis endpoint"
  value       = aws_elasticache_cluster.redis.cache_nodes[0].address
}
{{end}}

output "s3_bucket_name" {
  description = "S3 bucket name"
  value       = aws_s3_bucket.main.id
}
`))
}

// generateGCP creates GCP Terraform configuration (placeholder)
func (g *Generator) generateGCP(spec ArchitectureSpec) (map[string]string, error) {
	// TODO: Implement GCP templates
	return nil, fmt.Errorf("GCP generation not yet implemented")
}

// generateAzure creates Azure Terraform configuration (placeholder)
func (g *Generator) generateAzure(spec ArchitectureSpec) (map[string]string, error) {
	// TODO: Implement Azure templates
	return nil, fmt.Errorf("Azure generation not yet implemented")
}

// registerGCPTemplates registers GCP-specific templates (placeholder)
func (g *Generator) registerGCPTemplates() {
	// TODO: Add GCP templates
}

// registerAzureTemplates registers Azure-specific templates (placeholder)
func (g *Generator) registerAzureTemplates() {
	// TODO: Add Azure templates
}
