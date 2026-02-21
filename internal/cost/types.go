package cost

// CloudProvider represents a cloud platform
type CloudProvider string

const (
	ProviderAWS   CloudProvider = "aws"
	ProviderGCP   CloudProvider = "gcp"
	ProviderAzure CloudProvider = "azure"
)

// ArchitectureSpec describes the architecture to estimate
type ArchitectureSpec struct {
	Provider          string  `json:"provider"`
	Region            string  `json:"region"`
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
	HealthCheckPath   string  `json:"health_check_path"`
}

// ArchitectureCost contains total monthly cost breakdown
type ArchitectureCost struct {
	Provider           string            `json:"provider"`
	ComputeMonthly     float64           `json:"compute_monthly"`
	DatabaseMonthly    float64           `json:"database_monthly"`
	CacheMonthly       float64           `json:"cache_monthly"`
	NetworkMonthly     float64           `json:"network_monthly"`
	StorageMonthly     float64           `json:"storage_monthly"`
	TotalMonthly       float64           `json:"total_monthly"`
	HiddenCosts        []HiddenCost      `json:"hidden_costs"`
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
