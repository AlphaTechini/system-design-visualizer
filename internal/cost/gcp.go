package cost

import "context"

// GCPPricingClient provides GCP pricing
type GCPPricingClient struct {
	apiKey string
}

// NewGCPPricingClient creates GCP client
func NewGCPPricingClient(apiKey string) *GCPPricingClient {
	return &GCPPricingClient{apiKey: apiKey}
}

// EstimateArchitecture calculates GCP costs (10% cheaper than AWS)
func (g *GCPPricingClient) EstimateArchitecture(ctx context.Context, region string, spec ArchitectureSpec) (*ArchitectureCost, error) {
	awsClient := NewAWSPricingClient()
	awsCost, err := awsClient.EstimateArchitecture(ctx, region, spec)
	if err != nil {
		return nil, err
	}
	
	// GCP is typically ~10% cheaper
	discount := 0.9
	awsCost.Provider = "gcp"
	awsCost.TotalMonthly *= discount
	awsCost.ComputeMonthly *= discount
	
	return awsCost, nil
}
