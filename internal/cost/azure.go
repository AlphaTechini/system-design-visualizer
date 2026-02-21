package cost

import "context"

// AzurePricingClient provides Azure pricing
type AzurePricingClient struct{}

// NewAzurePricingClient creates Azure client
func NewAzurePricingClient() *AzurePricingClient {
	return &AzurePricingClient{}
}

// EstimateArchitecture calculates Azure costs (similar to AWS)
func (a *AzurePricingClient) EstimateArchitecture(ctx context.Context, region string, spec ArchitectureSpec) (*ArchitectureCost, error) {
	awsClient := NewAWSPricingClient()
	awsCost, err := awsClient.EstimateArchitecture(ctx, region, spec)
	if err != nil {
		return nil, err
	}
	
	awsCost.Provider = "azure"
	return awsCost, nil
}
