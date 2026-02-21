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

// generateGCP creates GCP Terraform configuration
func (g *Generator) generateGCP(spec ArchitectureSpec) (map[string]string, error) {
	files := make(map[string]string)
	
	// Generate main.tf
	mainTmpl, ok := g.templates["gcp_main"]
	if !ok {
		return nil, fmt.Errorf("template gcp_main not found")
	}
	
	var mainBuf bytes.Buffer
	if err := mainTmpl.Execute(&mainBuf, spec); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	files["main.tf"] = mainBuf.String()
	
	// Generate variables.tf
	variablesTmpl, ok := g.templates["gcp_variables"]
	if !ok {
		return nil, fmt.Errorf("template gcp_variables not found")
	}
	
	var variablesBuf bytes.Buffer
	if err := variablesTmpl.Execute(&variablesBuf, spec); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	files["variables.tf"] = variablesBuf.String()
	
	// Generate outputs.tf
	outputsTmpl, ok := g.templates["gcp_outputs"]
	if !ok {
		return nil, fmt.Errorf("template gcp_outputs not found")
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
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}
`)
	files["providers.tf"] = providersBuf.String()
	
	return files, nil
}

// generateAzure creates Azure Terraform configuration
func (g *Generator) generateAzure(spec ArchitectureSpec) (map[string]string, error) {
	files := make(map[string]string)
	
	// Generate main.tf
	mainTmpl, ok := g.templates["azure_main"]
	if !ok {
		return nil, fmt.Errorf("template azure_main not found")
	}
	
	var mainBuf bytes.Buffer
	if err := mainTmpl.Execute(&mainBuf, spec); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	files["main.tf"] = mainBuf.String()
	
	// Generate variables.tf
	variablesTmpl, ok := g.templates["azure_variables"]
	if !ok {
		return nil, fmt.Errorf("template azure_variables not found")
	}
	
	var variablesBuf bytes.Buffer
	if err := variablesTmpl.Execute(&variablesBuf, spec); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	files["variables.tf"] = variablesBuf.String()
	
	// Generate outputs.tf
	outputsTmpl, ok := g.templates["azure_outputs"]
	if !ok {
		return nil, fmt.Errorf("template azure_outputs not found")
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
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

provider "azurerm" {
  features {}
}
`)
	files["providers.tf"] = providersBuf.String()
	
	return files, nil
}

// registerGCPTemplates registers GCP-specific templates
func (g *Generator) registerGCPTemplates() {
	// GCP Main resources template
	g.templates["gcp_main"] = template.Must(template.New("gcp_main").Parse(`
# VPC Network
resource "google_compute_network" "vpc" {
  name                    = "${var.project_name}-vpc"
  auto_create_subnetworks = false
  routing_mode            = "REGIONAL"
}

resource "google_compute_subnetwork" "public" {
  count         = 2
  name          = "${var.project_name}-subnet-${count.index + 1}"
  ip_cidr_range = "10.0.${count.index + 1}.0/24"
  region        = var.region
  network       = google_compute_network.vpc.id
}

resource "google_compute_router" "router" {
  name    = "${var.project_name}-router"
  region  = var.region
  network = google_compute_network.vpc.id
  
  bgp {
    asn = 64514
  }
}

resource "google_compute_router_nat" "nat" {
  name                               = "${var.project_name}-nat"
  router                             = google_compute_router.router.name
  region                             = google_compute_router.router.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"
}

# Firewall Rules
resource "google_compute_firewall" "allow_http" {
  name    = "${var.project_name}-allow-http"
  network = google_compute_network.vpc.name
  
  allow {
    protocol = "tcp"
    ports    = ["80", "443"]
  }
  
  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["${var.project_name}-app"]
}

resource "google_compute_firewall" "allow_internal" {
  name    = "${var.project_name}-allow-internal"
  network = google_compute_network.vpc.name
  
  allow {
    protocol = "tcp"
    ports    = ["0-65535"]
  }
  
  allow {
    protocol = "udp"
    ports    = ["0-65535"]
  }
  
  source_ranges = ["10.0.0.0/16"]
}

# Instance Template
resource "google_compute_instance_template" "app" {
  name_prefix  = "${var.project_name}-"
  machine_type = var.instance_type
  
  disk {
    source_image = "debian-cloud/debian-11"
    auto_delete  = true
    boot         = true
  }
  
  network_interface {
    network    = google_compute_network.vpc.id
    subnetwork = google_compute_subnetwork.public[0].id
    access_config {
      # Ephemeral public IP
    }
  }
  
  metadata_startup_script = <<-EOF
              #!/bin/bash
              apt-get update
              apt-get install -y apache2
              systemctl start apache2
              systemctl enable apache2
              echo "<h1>${var.project_name}</h1>" > /var/www/html/index.html
              EOF
  
  tags = ["${var.project_name}-app"]
  
  lifecycle {
    create_before_destroy = true
  }
}

# Instance Group Manager
resource "google_compute_region_instance_group_manager" "app" {
  name     = "${var.project_name}-igm"
  region   = var.region
  
  version {
    instance_template = google_compute_instance_template.app.id
    name              = "primary"
  }
  
  base_instance_name = "${var.project_name}-vm"
  target_size        = {{.InstanceCount}}
  
  auto_healing_policies {
    health_check      = google_compute_health_check.app.id
    initial_delay_sec = 300
  }
}

# Health Check
resource "google_compute_health_check" "app" {
  name = "${var.project_name}-health-check"
  
  http_health_check {
    port               = 80
    request_path       = "{{.HealthCheckPath}}"
    check_interval_sec = 30
    timeout_sec        = 5
  }
}

# Load Balancer
resource "google_compute_forwarding_rule" "http" {
  name       = "${var.project_name}-http-fr"
  region     = var.region
  load_balancing_scheme = "EXTERNAL"
  backend_service = google_compute_region_backend_service.app.id
  port_range = "80"
  ip_protocol = "TCP"
}

resource "google_compute_region_backend_service" "app" {
  name          = "${var.project_name}-backend"
  region        = var.region
  protocol      = "HTTP"
  load_balancing_scheme = "EXTERNAL"
  health_checks = [google_compute_health_check.app.id]
  
  backend {
    group = google_compute_region_instance_group_manager.app.instance_group
  }
}

# Cloud SQL (PostgreSQL)
resource "google_sql_database_instance" "main" {
  name                = "${var.project_name}-db"
  database_version    = "POSTGRES_15"
  region              = var.region
  deletion_protection = false
  
  settings {
    tier              = "{{.DatabaseSize}}"
    availability_type = {{if .MultiAZ}}"REGIONAL"{{else}}"ZONAL"{{end}}
    
    disk_size_gb = {{.DatabaseStorageGB}}
    disk_type    = "PD_SSD"
    
    ip_configuration {
      ipv4_enabled = true
      require_ssl  = true
    }
    
    backup_configuration {
      enabled                        = true
      start_time                     = "02:00"
      point_in_time_recovery_enabled = true
    }
  }
  
  database {
    name = "appdb"
  }
  
  user {
    name     = "admin"
    password = var.db_password
  }
}

# Memorystore (Redis)
{{if .CacheNodes}}
resource "google_redis_instance" "cache" {
  name           = "${var.project_name}-redis"
  tier           = "STANDARD_HA"
  memory_size_gb = 1
  region         = var.region
  
  authorized_network = google_compute_network.vpc.id
  connect_mode       = "DIRECT_PEERING"
}
{{end}}

# Cloud Storage Bucket
resource "google_storage_bucket" "main" {
  name          = "${var.project_name}-bucket"
  location      = var.region
  force_destroy = true
  
  versioning {
    enabled = true
  }
  
  uniform_bucket_level_access = true
}

# Monitoring Alerts
{{if .EnableMonitoring}}
resource "google_monitoring_alert_policy" "cpu_high" {
  display_name = "${var.project_name} CPU High"
  
  combiner = "OR"
  
  conditions {
    display_name = "CPU utilization > 80%"
    
    condition_threshold {
      filter          = "resource.type=\"gce_instance\" AND metric.type=\"compute.googleapis.com/instance/cpu/utilization\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0.8
      
      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }
  
  notification_channels = [google_monitoring_notification_channel.email.id]
}

resource "google_monitoring_notification_channel" "email" {
  display_name = "${var.project_name} Email Alert"
  type         = "email"
  
  labels = {
    email_address = "{{.AlertEmail}}"
  }
}
{{end}}
`))

	g.templates["gcp_variables"] = template.Must(template.New("gcp_variables").Parse(`
variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP Region"
  type        = string
  default     = "us-central1"
}

variable "project_name" {
  description = "Project name for resource naming"
  type        = string
}

variable "instance_type" {
  description = "GCP machine type"
  type        = string
  default     = "e2-micro"
}

variable "db_password" {
  description = "Cloud SQL admin password"
  type        = string
  sensitive   = true
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}
`))

	g.templates["gcp_outputs"] = template.Must(template.New("gcp_outputs").Parse(`
output "vpc_id" {
  description = "VPC Network ID"
  value       = google_compute_network.vpc.id
}

output "load_balancer_ip" {
  description = "Load Balancer IP address"
  value       = google_compute_forwarding_rule.http.ip_address
}

output "cloud_sql_connection_name" {
  description = "Cloud SQL connection name"
  value       = google_sql_database_instance.main.connection_name
}

{{if .CacheNodes}}
output "redis_host" {
  description = "Memorystore Redis host"
  value       = google_redis_instance.cache.host
}
{{end}}

output "storage_bucket_name" {
  description = "Cloud Storage bucket name"
  value       = google_storage_bucket.main.name
}
`))
}

// registerAzureTemplates registers Azure-specific templates
func (g *Generator) registerAzureTemplates() {
	// Azure Main resources template
	g.templates["azure_main"] = template.Must(template.New("azure_main").Parse(`
# Resource Group
resource "azurerm_resource_group" "main" {
  name     = "${var.project_name}-rg"
  location = var.location
}

# Virtual Network
resource "azurerm_virtual_network" "main" {
  name                = "${var.project_name}-vnet"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  address_space       = ["10.0.0.0/16"]
}

resource "azurerm_subnet" "public" {
  count                = 2
  name                 = "${var.project_name}-subnet-${count.index + 1}"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.${count.index + 1}.0/24"]
}

# Network Security Group
resource "azurerm_network_security_group" "app" {
  name                = "${var.project_name}-app-nsg"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  
  security_rule {
    name                       = "AllowHTTP"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "80"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
  
  security_rule {
    name                       = "AllowHTTPS"
    priority                   = 110
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

# Public IP for Load Balancer
resource "azurerm_public_ip" "lb" {
  name                = "${var.project_name}-lb-pip"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  allocation_method   = "Static"
  sku                 = "Standard"
}

# Load Balancer
resource "azurerm_lb" "app" {
  name                = "${var.project_name}-lb"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = "Standard"
  
  frontend_ip_configuration {
    name                 = "PublicIPAddress"
    public_ip_address_id = azurerm_public_ip.lb.id
  }
}

resource "azurerm_lb_backend_address_pool" "app" {
  name            = "${var.project_name}-bepool"
  loadbalancer_id = azurerm_lb.app.id
}

resource "azurerm_lb_probe" "app" {
  name                = "${var.project_name}-probe"
  loadbalancer_id     = azurerm_lb.app.id
  protocol            = "Http"
  port                = 80
  request_path        = "{{.HealthCheckPath}}"
  interval_in_seconds = 30
  number_of_probes    = 2
}

resource "azurerm_lb_rule" "app" {
  name                           = "${var.project_name}-httprule"
  loadbalancer_id                = azurerm_lb.app.id
  protocol                         = "Tcp"
  frontend_port                    = 80
  backend_port                     = 80
  frontend_ip_configuration_name   = "PublicIPAddress"
  backend_address_pool_ids         = [azurerm_lb_backend_address_pool.app.id]
  probe_id                         = azurerm_lb_probe.app.id
  idle_timeout_in_minutes          = 4
  enable_floating_ip               = false
  load_distribution                = "Default"
  disable_outbound_snat            = false
  enable_tcp_reset                 = false
  disable_outbound_tcp_reuse       = false
}

# Availability Set
resource "azurerm_availability_set" "app" {
  name                         = "${var.project_name}-avset"
  location                     = azurerm_resource_group.main.location
  resource_group_name          = azurerm_resource_group.main.name
  platform_fault_domain_count  = 2
  platform_update_domain_count = 5
  managed                      = true
}

# VM Scale Set
resource "azurerm_linux_virtual_machine_scale_set" "app" {
  name                = "${var.project_name}-vmss"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = var.instance_type
  instances           = {{.InstanceCount}}
  admin_username      = "azureuser"
  
  availability_set_id = azurerm_availability_set.app.id
  
  admin_ssh_key {
    username   = "azureuser"
    public_key = var.ssh_public_key
  }
  
  source_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "18.04-LTS"
    version   = "latest"
  }
  
  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }
  
  network_interface {
    name    = "primary"
    primary = true
    
    ip_configuration {
      name                                   = "internal"
      primary                                = true
      subnet_id                              = azurerm_subnet.public[0].id
      load_balancer_backend_address_pool_ids = [azurerm_lb_backend_address_pool.app.id]
    }
  }
  
  custom_data = base64encode(<<-EOF
              #cloud-config
              package_update: true
              packages:
                - apache2
              runcmd:
                - systemctl start apache2
                - systemctl enable apache2
                - echo "<h1>${var.project_name}</h1>" > /var/www/html/index.html
              EOF
  )
}

# Azure Database for PostgreSQL
resource "azurerm_postgresql_server" "main" {
  name                = "${var.project_name}-db"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  
  sku_name   = "{{.DatabaseSize}}"
  storage_mb = {{mul .DatabaseStorageGB 1024}}
  version    = "11"
  
  administrator_login          = "admin"
  administrator_login_password = var.db_password
  
  ssl_enforcement_enabled          = true
  ssl_minimal_tls_version_enforced = "TLS1_2"
  
  geo_redundant_backup_enabled = {{.MultiAZ}}
  auto_grow_enabled            = true
  backup_retention_days        = 7
}

resource "azurerm_postgresql_firewall_rule" "allow_all" {
  name                = "AllowAll"
  resource_group_name = azurerm_resource_group.main.name
  server_name         = azurerm_postgresql_server.main.name
  start_ip_address    = "0.0.0.0"
  end_ip_address      = "255.255.255.255"
}

# Azure Cache for Redis
{{if .CacheNodes}}
resource "azurerm_redis_cache" "main" {
  name                = "${var.project_name}-redis"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  capacity            = {{.CacheNodes}}
  family              = "C"
  sku_name            = "Basic"
  enable_non_ssl_port = false
  minimum_tls_version = "1.2"
}
{{end}}

# Storage Account (Blob)
resource "azurerm_storage_account" "main" {
  name                     = "${replace(var.project_name, "-", "")}sa"
  location                 = azurerm_resource_group.main.location
  resource_group_name      = azurerm_resource_group.main.name
  account_tier             = "Standard"
  account_replication_type = "LRS"
  
  blob_properties {
    versioning_enabled = true
  }
}

# Monitor Alerts
{{if .EnableMonitoring}}
resource "azurerm_monitor_action_group" "email" {
  name                = "${var.project_name}-ag"
  resource_group_name = azurerm_resource_group.main.name
  short_name          = "${replace(var.project_name, "-", "")}AG"
  
  email_receiver {
    name          = "sendtoadmin"
    email_address = "{{.AlertEmail}}"
  }
}

resource "azurerm_monitor_metric_alert" "cpu_high" {
  name                = "${var.project_name}-cpu-alert"
  resource_group_name = azurerm_resource_group.main.name
  scopes              = [azurerm_linux_virtual_machine_scale_set.app.id]
  description         = "CPU usage is too high"
  
  criteria {
    metric_namespace = "Microsoft.Compute/virtualMachineScaleSets"
    metric_name      = "Percentage CPU"
    operator         = "GreaterThan"
    threshold        = 80
  }
  
  action {
    action_group_id = azurerm_monitor_action_group.email.id
  }
}
{{end}}
`))

	g.templates["azure_variables"] = template.Must(template.New("azure_variables").Parse(`
variable "location" {
  description = "Azure Region"
  type        = string
  default     = "East US"
}

variable "project_name" {
  description = "Project name for resource naming"
  type        = string
}

variable "instance_type" {
  description = "Azure VM size"
  type        = string
  default     = "Standard_B1s"
}

variable "ssh_public_key" {
  description = "SSH public key for VM access"
  type        = string
}

variable "db_password" {
  description = "PostgreSQL admin password"
  type        = string
  sensitive   = true
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}
`))

	g.templates["azure_outputs"] = template.Must(template.New("azure_outputs").Parse(`
output "resource_group_name" {
  description = "Resource Group name"
  value       = azurerm_resource_group.main.name
}

output "load_balancer_ip" {
  description = "Load Balancer public IP"
  value       = azurerm_public_ip.lb.ip_address
}

output "postgresql_fqdn" {
  description = "PostgreSQL server FQDN"
  value       = azurerm_postgresql_server.main.fqdn
}

{{if .CacheNodes}}
output "redis_hostname" {
  description = "Redis Cache hostname"
  value       = azurerm_redis_cache.main.hostname
}
{{end}}

output "storage_account_name" {
  description = "Storage Account name"
  value       = azurerm_storage_account.main.name
}
`))
}
