package cost

import (
	"context"
	"fmt"
	"time"
)

// CostEstimator combines pricing from all cloud providers
type CostEstimator struct {
	awsClient   *AWSPricingClient
	gcpClient   *GCPPricingClient
	azureClient *AzurePricingClient
}

// NewCostEstimator creates estimator with API clients
func NewCostEstimator(gcpAPIKey string) *CostEstimator {
	return &CostEstimator{
		awsClient:   NewAWSPricingClient(),
		gcpClient:   NewGCPPricingClient(gcpAPIKey),
		azureClient: NewAzurePricingClient(),
	}
}

// ArchitectureCost contains total monthly cost breakdown
type ArchitectureCost struct {
	Provider        string            `json:"provider"`
	ComputeMonthly  float64           `json:"compute_monthly"`
	DatabaseMonthly float64           `json:"database_monthly"`
	CacheMonthly    float64           `json:"cache_monthly"`
	NetworkMonthly  float64           `json:"network_monthly"`
	StorageMonthly  float64           `json:"storage_monthly"`
	TotalMonthly    float64           `json:"total_monthly"`
	HiddenCosts     []HiddenCost      `json:"hidden_costs"`
	ScalingProjections ScalingProjection `json:"scaling_projections"`
}

// HiddenCost represents often-overlooked costs
type HiddenCost struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	MonthlyCost float64 `json:"monthly_cost"`
}

// ScalingProjection shows cost at different scales
type ScalingProjection struct {
	CurrentUsers  int     `json:"current_users"`
	CurrentCost   float64 `json:"current_cost"`
	TenXUsers     int     `json:"ten_x_users"`
	TenXCost      float64 `json:"ten_x_cost"`
	HundredXUsers int     `json:"hundred_x_users"`
	HundredXCost  float64 `json:"hundred_x_cost"`
}

// EstimateArchitecture calculates total monthly cost for an architecture
func (e *CostEstimator) EstimateArchitecture(ctx context.Context, provider CloudProvider, region string, arch ArchitectureSpec) (*ArchitectureCost, error) {
	switch provider {
	case ProviderAWS:
		return e.estimateAWS(ctx, region, arch)
	case ProviderGCP:
		return e.estimateGCP(ctx, region, arch)
	case ProviderAzure:
		return e.estimateAzure(ctx, region, arch)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// ArchitectureSpec describes the architecture to estimate
type ArchitectureSpec struct {
	InstanceType      string  `json:"instance_type"`
	InstanceCount     int     `json:"instance_count"`
	DatabaseType      string  `json:"database_type"`
	DatabaseSize      string  `json:"database_size"`
	CacheType         string  `json:"cache_type"`
	CacheNodes        int     `json:"cache_nodes"`
	DataTransferOutGB float64 `json:"data_transfer_out_gb"`
	StorageGB         float64 `json:"storage_gb"`
	StorageRequests   int64   `json:"storage_requests"`
	CrossAZ           bool    `json:"cross_az"`
	ExpectedUsers     int     `json:"expected_users"`
}

// estimateAWS calculates AWS-specific costs
func (e *CostEstimator) estimateAWS(ctx context.Context, region string, arch ArchitectureSpec) (*ArchitectureCost, error) {
	// Get EC2 pricing
	ec2Prices, err := e.awsClient.GetProducts(ctx, "AmazonEC2", map[string]string{})
	if err != nil {
		return nil, fmt.Errorf("get EC2 pricing: %w", err)
	}
	
	// Calculate compute cost (simplified - match instance type)
	var hourlyRate float64 = 0.1 // Default fallback
	for _, product := range ec2Prices {
		if product.Attributes.InstanceType == arch.InstanceType {
			// Extract price from terms (simplified)
			hourlyRate = 0.192 // m5.large default
			break
		}
	}
	
	computeMonthly := hourlyRate * 24 * 30 * float64(arch.InstanceCount)
	
	// Get RDS pricing
	rdsPrice, err := e.awsClient.GetRDSPricing(ctx, region, arch.DatabaseType, arch.DatabaseSize)
	if err != nil {
		return nil, fmt.Errorf("get RDS pricing: %w", err)
	}
	databaseMonthly := rdsPrice.MonthlyCost + rdsPrice.BackupCost
	
	// Get ElastiCache pricing
	cachePrice, err := e.awsClient.GetElastiCachePricing(ctx, region, arch.CacheType)
	if err != nil {
		return nil, fmt.Errorf("get ElastiCache pricing: %w", err)
	}
	cacheMonthly := cachePrice.MonthlyCost * float64(arch.CacheNodes)
	
	// Calculate network costs
	networkPrice, err := e.awsClient.CalculateDataTransferCost(ctx, ProviderAWS, region, arch.DataTransferOutGB, arch.CrossAZ)
	if err != nil {
		return nil, fmt.Errorf("get network pricing: %w", err)
	}
	networkMonthly := networkPrice.TotalCost
	
	// Calculate S3 storage costs
	storageMonthly, err := e.awsClient.GetS3Pricing(ctx, region, arch.StorageGB, arch.StorageRequests)
	if err != nil {
		return nil, fmt.Errorf("get S3 pricing: %w", err)
	}
	
	totalMonthly := computeMonthly + databaseMonthly + cacheMonthly + networkMonthly + storageMonthly
	
	// Identify hidden costs
	hiddenCosts := []HiddenCost{}
	
	if arch.CrossAZ {
		hiddenCosts = append(hiddenCosts, HiddenCost{
			Name:        "Cross-AZ Data Transfer",
			Description: "Data transfer between availability zones ($0.01/GB)",
			MonthlyCost: arch.DataTransferOutGB * 0.01,
		})
	}
	
	if arch.DatabaseType == "oracle-ee" || arch.DatabaseType == "sqlserver-ee" {
		hiddenCosts = append(hiddenCosts, HiddenCost{
			Name:        "Enterprise License Premium",
			Description: "Oracle/SQL Server Enterprise edition licensing (3x base cost)",
			MonthlyCost: databaseMonthly * 2.0, // Additional 200% on top of base
		})
	}
	
	// Calculate scaling projections
	currentPerUserCost := totalMonthly / float64(max(1, arch.ExpectedUsers))
	
	return &ArchitectureCost{
		Provider:        "aws",
		ComputeMonthly:  computeMonthly,
		DatabaseMonthly: databaseMonthly,
		CacheMonthly:    cacheMonthly,
		NetworkMonthly:  networkMonthly,
		StorageMonthly:  storageMonthly,
		TotalMonthly:    totalMonthly,
		HiddenCosts:     hiddenCosts,
		ScalingProjections: ScalingProjection{
			CurrentUsers:  arch.ExpectedUsers,
			CurrentCost:   totalMonthly,
			TenXUsers:     arch.ExpectedUsers * 10,
			TenXCost:      totalMonthly * 8, // Economies of scale (8x not 10x)
			HundredXUsers: arch.ExpectedUsers * 100,
			HundredXCost:  totalMonthly * 60, // Better economies at scale
		},
	}, nil
}

// estimateGCP calculates GCP-specific costs (placeholder - integrate real API)
func (e *CostEstimator) estimateGCP(ctx context.Context, region string, arch ArchitectureSpec) (*ArchitectureCost, error) {
	// TODO: Integrate with GCP Pricing API
	// For now, return AWS equivalent with 10% discount (GCP is typically cheaper)
	awsCost, err := e.estimateAWS(ctx, region, arch)
	if err != nil {
		return nil, err
	}
	
	awsCost.Provider = "gcp"
	awsCost.TotalMonthly *= 0.9 // 10% GCP discount
	awsCost.ComputeMonthly *= 0.9
	awsCost.DatabaseMonthly *= 0.9
	
	return awsCost, nil
}

// estimateAzure calculates Azure-specific costs (placeholder - integrate real API)
func (e *CostEstimator) estimateAzure(ctx context.Context, region string, arch ArchitectureSpec) (*ArchitectureCost, error) {
	// TODO: Integrate with Azure Retail Prices API
	// For now, return AWS equivalent
	awsCost, err := e.estimateAWS(ctx, region, arch)
	if err != nil {
		return nil, err
	}
	
	awsCost.Provider = "azure"
	
	return awsCost, nil
}

// CompareProviders compares costs across all cloud providers
func (e *CostEstimator) CompareProviders(ctx context.Context, region string, arch ArchitectureSpec) (map[string]*ArchitectureCost, error) {
	results := make(map[string]*ArchitectureCost)
	
	// AWS
	awsCost, err := e.EstimateArchitecture(ctx, ProviderAWS, region, arch)
	if err != nil {
		return nil, fmt.Errorf("AWS estimate failed: %w", err)
	}
	results["aws"] = awsCost
	
	// GCP
	gcpCost, err := e.EstimateArchitecture(ctx, ProviderGCP, region, arch)
	if err != nil {
		return nil, fmt.Errorf("GCP estimate failed: %w", err)
	}
	results["gcp"] = gcpCost
	
	// Azure
	azureCost, err := e.EstimateArchitecture(ctx, ProviderAzure, region, arch)
	if err != nil {
		return nil, fmt.Errorf("Azure estimate failed: %w", err)
	}
	results["azure"] = azureCost
	
	return results, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
