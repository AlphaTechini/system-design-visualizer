package cost

import "context"

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

// EstimateArchitecture calculates cost for specified provider
func (e *CostEstimator) EstimateArchitecture(ctx context.Context, provider CloudProvider, region string, spec ArchitectureSpec) (*ArchitectureCost, error) {
	switch provider {
	case ProviderAWS:
		return e.awsClient.EstimateArchitecture(ctx, region, spec)
	case ProviderGCP:
		return e.gcpClient.EstimateArchitecture(ctx, region, spec)
	case ProviderAzure:
		return e.azureClient.EstimateArchitecture(ctx, region, spec)
	default:
		return nil, nil
	}
}

// CompareProviders compares costs across all clouds
func (e *CostEstimator) CompareProviders(ctx context.Context, region string, spec ArchitectureSpec) (map[string]*ArchitectureCost, error) {
	results := make(map[string]*ArchitectureCost)
	
	awsCost, err := e.EstimateArchitecture(ctx, ProviderAWS, region, spec)
	if err != nil {
		return nil, err
	}
	results["aws"] = awsCost
	
	gcpCost, err := e.EstimateArchitecture(ctx, ProviderGCP, region, spec)
	if err != nil {
		return nil, err
	}
	results["gcp"] = gcpCost
	
	azureCost, err := e.EstimateArchitecture(ctx, ProviderAzure, region, spec)
	if err != nil {
		return nil, err
	}
	results["azure"] = azureCost
	
	return results, nil
}
