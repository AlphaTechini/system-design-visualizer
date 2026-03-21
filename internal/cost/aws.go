package cost

import "context"

// AWSPricingClient provides AWS pricing
type AWSPricingClient struct{}

// NewAWSPricingClient creates AWS client
func NewAWSPricingClient() *AWSPricingClient {
	return &AWSPricingClient{}
}

// EstimateArchitecture calculates AWS costs
func (a *AWSPricingClient) EstimateArchitecture(ctx context.Context, region string, spec ArchitectureSpec) (*ArchitectureCost, error) {
	// Simplified pricing for MVP
	computeHourly := 0.192 // m5.large
	computeMonthly := computeHourly * 24 * 30 * float64(spec.InstanceCount)
	
	databaseMonthly := 0.017 * 24 * 30 // db.t3.micro
	cacheMonthly := 0.017 * 24 * 30 * float64(spec.CacheNodes)
	networkMonthly := spec.DataTransferOutGB * 0.09
	storageMonthly := spec.StorageGB * 0.023
	
	totalMonthly := computeMonthly + databaseMonthly + cacheMonthly + networkMonthly + storageMonthly
	
	var hiddenCosts []HiddenCost
	if spec.CrossAZ {
		hiddenCosts = append(hiddenCosts, HiddenCost{
			Name: "Cross-AZ Data Transfer",
			Description: "$0.01/GB between availability zones",
			MonthlyCost: spec.DataTransferOutGB * 0.01,
		})
	}
	
	return &ArchitectureCost{
		Provider: "aws",
		ComputeMonthly: computeMonthly,
		DatabaseMonthly: databaseMonthly,
		CacheMonthly: cacheMonthly,
		NetworkMonthly: networkMonthly,
		StorageMonthly: storageMonthly,
		TotalMonthly: totalMonthly,
		HiddenCosts: hiddenCosts,
		ScalingProjections: ScalingProjection{
			CurrentUsers: spec.ExpectedUsers,
			CurrentCost: totalMonthly,
			TenXUsers: spec.ExpectedUsers * 10,
			TenXCost: totalMonthly * 8,
			HundredXUsers: spec.ExpectedUsers * 100,
			HundredXCost: totalMonthly * 60,
		},
	}, nil
}
