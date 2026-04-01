package cost

import (
	"context"
	"math"
	"testing"
)

func TestAWSEstimateArchitecture(t *testing.T) {
	client := NewAWSPricingClient()
	spec := ArchitectureSpec{
		InstanceCount:   2,
		CacheNodes:      1,
		DataTransferOutGB: 100,
		StorageGB:       50,
		ExpectedUsers:   1000,
		CrossAZ:         true,
	}

	cost, err := client.EstimateArchitecture(context.Background(), "us-east-1", spec)
	if err != nil {
		t.Fatalf("EstimateArchitecture failed: %v", err)
	}

	if cost.Provider != "aws" {
		t.Errorf("Expected provider aws, got %s", cost.Provider)
	}

	// Verify monthly costs (based on logic in aws.go)
	// computeHourly := 0.192; 0.192 * 24 * 30 * 2 = 276.48
	epsilon := 0.001
	expectedCompute := 0.192 * 24 * 30 * 2
	if math.Abs(cost.ComputeMonthly-expectedCompute) > epsilon {
		t.Errorf("Expected ComputeMonthly %f, got %f", expectedCompute, cost.ComputeMonthly)
	}

	// Check hidden costs
	if len(cost.HiddenCosts) == 0 {
		t.Error("Expected hidden costs when CrossAZ is true")
	}
	foundCrossAZ := false
	for _, hc := range cost.HiddenCosts {
		if hc.Name == "Cross-AZ Data Transfer" {
			foundCrossAZ = true
			expectedHC := 100 * 0.01
			if math.Abs(hc.MonthlyCost-expectedHC) > epsilon {
				t.Errorf("Expected hidden cost %f, got %f", expectedHC, hc.MonthlyCost)
			}
		}
	}
	if !foundCrossAZ {
		t.Error("Did not find Cross-AZ Data Transfer hidden cost")
	}

	// Scaling projections
	if cost.ScalingProjections.CurrentUsers != 1000 {
		t.Errorf("Expected 1000 users, got %d", cost.ScalingProjections.CurrentUsers)
	}
	if cost.ScalingProjections.TenXUsers != 10000 {
		t.Errorf("Expected 10000 users, got %d", cost.ScalingProjections.TenXUsers)
	}
}

func TestCostEstimator_EstimateArchitecture(t *testing.T) {
	e := NewCostEstimator("gcp-key")
	ctx := context.Background()
	spec := ArchitectureSpec{InstanceCount: 1}

	// Test AWS
	awsCost, err := e.EstimateArchitecture(ctx, ProviderAWS, "us-east-1", spec)
	if err != nil || awsCost == nil || awsCost.Provider != "aws" {
		t.Errorf("AWS estimation failed: %v", err)
	}

	// Test GCP
	gcpCost, err := e.EstimateArchitecture(ctx, ProviderGCP, "us-east-1", spec)
	if err != nil || gcpCost == nil || gcpCost.Provider != "gcp" {
		t.Errorf("GCP estimation failed: %v", err)
	}

	// Test Azure
	azureCost, err := e.EstimateArchitecture(ctx, ProviderAzure, "us-east-1", spec)
	if err != nil || azureCost == nil || azureCost.Provider != "azure" {
		t.Errorf("Azure estimation failed: %v", err)
	}

	// Test Default
	unknownCost, err := e.EstimateArchitecture(ctx, "unknown", "us-east-1", spec)
	if err != nil || unknownCost != nil {
		t.Errorf("Expected nil for unknown provider, got %v", unknownCost)
	}
}

func TestCostEstimator_CompareProviders(t *testing.T) {
	e := NewCostEstimator("gcp-key")
	ctx := context.Background()
	spec := ArchitectureSpec{InstanceCount: 1}

	results, err := e.CompareProviders(ctx, "us-east-1", spec)
	if err != nil {
		t.Fatalf("CompareProviders failed: %v", err)
	}

	providers := []string{"aws", "gcp", "azure"}
	for _, p := range providers {
		if _, ok := results[p]; !ok {
			t.Errorf("Provider %s missing from results", p)
		}
	}
}
