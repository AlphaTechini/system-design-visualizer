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
