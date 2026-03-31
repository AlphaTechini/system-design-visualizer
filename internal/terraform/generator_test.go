package terraform

import (
	"strings"
	"testing"
)

func TestNewGenerator(t *testing.T) {
	g := NewGenerator()
	if g == nil {
		t.Fatal("NewGenerator returned nil")
	}
	if len(g.templates) == 0 {
		t.Error("NewGenerator did not initialize templates")
	}
}

func TestGenerate_AWS(t *testing.T) {
	g := NewGenerator()
	spec := ArchitectureSpec{
		Provider:        "aws",
		Region:          "us-east-1",
		ProjectName:     "test-project",
		InstanceCount:   2,
		DatabaseEngine:  "postgres",
		DatabaseSize:    "db.t3.micro",
		DatabaseStorageGB: 20,
		HealthCheckPath: "/health",
	}

	files, err := g.Generate(spec)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expectedFiles := []string{"main.tf", "variables.tf", "outputs.tf", "providers.tf"}
	for _, fileName := range expectedFiles {
		if _, ok := files[fileName]; !ok {
			t.Errorf("Expected file %s not found in output", fileName)
		}
	}

	// Verify some content
	if mainTF, ok := files["main.tf"]; ok {
		if !strings.Contains(mainTF, "Name = \"${var.project_name}-vpc\"") {
			t.Errorf("main.tf does not contain expected project name reference")
		}
		if !strings.Contains(mainTF, "max_size         = 2") {
			// Looking at the template in generator.go:
			// max_size         = {{.InstanceCount}}
			// desired_capacity = {{.InstanceCount}}
			t.Errorf("main.tf does not contain correct instance count")
		}
	}
}

func TestGenerate_AWS_WithCacheAndMonitoring(t *testing.T) {
	g := NewGenerator()
	spec := ArchitectureSpec{
		Provider:         "aws",
		Region:           "us-east-1",
		ProjectName:      "test-project",
		CacheEngine:      "redis",
		CacheNodeType:    "cache.t3.micro",
		CacheNodes:       1,
		EnableMonitoring: true,
		AlertEmail:       "admin@example.com",
	}

	files, err := g.Generate(spec)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	mainTF := files["main.tf"]
	if !strings.Contains(mainTF, "aws_elasticache_cluster") {
		t.Error("main.tf should contain ElastiCache resource")
	}
	if !strings.Contains(mainTF, "aws_cloudwatch_metric_alarm") {
		t.Error("main.tf should contain CloudWatch alarm")
	}
	if !strings.Contains(mainTF, "admin@example.com") {
		t.Error("main.tf should contain alert email")
	}
}

func TestGenerate_UnsupportedProvider(t *testing.T) {
	g := NewGenerator()
	spec := ArchitectureSpec{
		Provider: "unsupported",
	}

	_, err := g.Generate(spec)
	if err == nil {
		t.Error("Generate should fail for unsupported provider")
	}
}

func TestGenerate_PlaceholderProviders(t *testing.T) {
	g := NewGenerator()

	gcpSpec := ArchitectureSpec{Provider: "gcp"}
	_, err := g.Generate(gcpSpec)
	if err == nil || err.Error() != "GCP generation not yet implemented" {
		t.Errorf("Expected GCP implementation error, got %v", err)
	}

	azureSpec := ArchitectureSpec{Provider: "azure"}
	_, err = g.Generate(azureSpec)
	if err == nil || err.Error() != "Azure generation not yet implemented" {
		t.Errorf("Expected Azure implementation error, got %v", err)
	}
}
