package cost

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CloudProvider represents a cloud platform
type CloudProvider string

const (
	ProviderAWS   CloudProvider = "aws"
	ProviderGCP   CloudProvider = "gcp"
	ProviderAzure CloudProvider = "azure"
)

// PricingClient fetches cloud pricing data
type PricingClient struct {
	httpClient *http.Client
	cache      *MemoryCache
}

// InstancePricing represents compute instance costs
type InstancePricing struct {
	InstanceType    string  `json:"instance_type"`
	VCPU            int     `json:"vcpu"`
	MemoryGB        float64 `json:"memory_gb"`
	OnDemandHourly  float64 `json:"on_demand_hourly"`
	ReservedYearly  float64 `json:"reserved_yearly"`
	SpotHourly      float64 `json:"spot_hourly,omitempty"`
	Provider        string  `json:"provider"`
	Region          string  `json:"region"`
}

// DatabasePricing represents managed database costs
type DatabasePricing struct {
	InstanceType    string  `json:"instance_type"`
	Engine          string  `json:"engine"` // postgres, mysql, mongodb
	StorageGB       float64 `json:"storage_gb"`
	IOPS            int     `json:"iops"`
	MonthlyCost     float64 `json:"monthly_cost"`
	BackupCost      float64 `json:"backup_cost"`
	Provider        string  `json:"provider"`
}

// NetworkPricing represents data transfer costs
type NetworkPricing struct {
	DataTransferOutGB float64 `json:"data_transfer_out_gb"`
	CostPerGB         float64 `json:"cost_per_gb"`
	TotalCost         float64 `json:"total_cost"`
	CrossAZ           bool    `json:"cross_az"`
	Provider          string  `json:"provider"`
}

// NewPricingClient creates pricing client with caching
func NewPricingClient() *PricingClient {
	return &PricingClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: NewMemoryCache(1 * time.Hour),
	}
}

// GetEC2Pricing fetches AWS EC2 instance pricing
func (p *PricingClient) GetEC2Pricing(ctx context.Context, region string, instanceTypes []string) ([]InstancePricing, error) {
	cacheKey := fmt.Sprintf("aws:ec2:%s:%v", region, instanceTypes)
	
	if cached, ok := p.cache.Get(cacheKey); ok {
		return cached.([]InstancePricing), nil
	}

	// Note: In production, integrate with AWS Price List API
	// For MVP, using static pricing table (updated quarterly)
	pricings := []InstancePricing{
		{
			InstanceType:   "t3.micro",
			VCPU:           2,
			MemoryGB:       1,
			OnDemandHourly: 0.0104,
			ReservedYearly: 0.0065,
			SpotHourly:     0.0042,
			Provider:       "aws",
			Region:         region,
		},
		{
			InstanceType:   "t3.small",
			VCPU:           2,
			MemoryGB:       2,
			OnDemandHourly: 0.0208,
			ReservedYearly: 0.013,
			SpotHourly:     0.0084,
			Provider:       "aws",
			Region:         region,
		},
		{
			InstanceType:   "t3.medium",
			VCPU:           2,
			MemoryGB:       4,
			OnDemandHourly: 0.0416,
			ReservedYearly: 0.026,
			SpotHourly:     0.0168,
			Provider:       "aws",
			Region:         region,
		},
		{
			InstanceType:   "m5.large",
			VCPU:           2,
			MemoryGB:       8,
			OnDemandHourly: 0.096,
			ReservedYearly: 0.06,
			SpotHourly:     0.0384,
			Provider:       "aws",
			Region:         region,
		},
		{
			InstanceType:   "m5.xlarge",
			VCPU:           4,
			MemoryGB:       16,
			OnDemandHourly: 0.192,
			ReservedYearly: 0.12,
			SpotHourly:     0.0768,
			Provider:       "aws",
			Region:         region,
		},
		{
			InstanceType:   "c5.large",
			VCPU:           2,
			MemoryGB:       4,
			OnDemandHourly: 0.085,
			ReservedYearly: 0.053,
			SpotHourly:     0.034,
			Provider:       "aws",
			Region:         region,
		},
		{
			InstanceType:   "r5.large",
			VCPU:           2,
			MemoryGB:       16,
			OnDemandHourly: 0.126,
			ReservedYearly: 0.079,
			SpotHourly:     0.0504,
			Provider:       "aws",
			Region:         region,
		},
	}

	// Filter to requested types
	if len(instanceTypes) > 0 {
		filtered := []InstancePricing{}
		for _, p := range pricings {
			for _, t := range instanceTypes {
				if p.InstanceType == t {
					filtered = append(filtered, p)
				}
			}
		}
		pricings = filtered
	}

	p.cache.Set(cacheKey, pricings, 24*time.Hour)
	return pricings, nil
}

// GetRDSPricing fetches AWS RDS pricing
func (p *PricingClient) GetRDSPricing(ctx context.Context, region string, engine string, instanceType string) (*DatabasePricing, error) {
	cacheKey := fmt.Sprintf("aws:rds:%s:%s:%s", region, engine, instanceType)
	
	if cached, ok := p.cache.Get(cacheKey); ok {
		return cached.(*DatabasePricing), nil
	}

	// Static pricing for MVP (update quarterly)
	basePrices := map[string]float64{
		"db.t3.micro":   0.017,
		"db.t3.small":   0.034,
		"db.t3.medium":  0.068,
		"db.m5.large":   0.171,
		"db.m5.xlarge":  0.342,
		"db.r5.large":   0.24,
		"db.r5.xlarge":  0.48,
	}

	hourlyRate, exists := basePrices[instanceType]
	if !exists {
		hourlyRate = 0.1 // Default fallback
	}

	// Engine multiplier (PostgreSQL is baseline, Oracle/SQL Server cost more)
	engineMultiplier := 1.0
	switch engine {
	case "oracle-ee", "sqlserver-ee":
		engineMultiplier = 3.0
	case "oracle-se2", "sqlserver-se":
		engineMultiplier = 2.0
	case "mysql", "mariadb":
		engineMultiplier = 0.9
	}

	monthlyCost := hourlyRate * engineMultiplier * 24 * 30
	backupCost := monthlyCost * 0.2 // 20% of DB cost for backups

	pricing := &DatabasePricing{
		InstanceType: instanceType,
		Engine:       engine,
		StorageGB:    100, // Default
		IOPS:         3000, // General purpose SSD
		MonthlyCost:  monthlyCost,
		BackupCost:   backupCost,
		Provider:     "aws",
	}

	p.cache.Set(cacheKey, pricing, 24*time.Hour)
	return pricing, nil
}

// GetElastiCachePricing fetches AWS ElastiCache (Redis/Memcached) pricing
func (p *PricingClient) GetElastiCachePricing(ctx context.Context, region string, nodeType string) (*DatabasePricing, error) {
	cacheKey := fmt.Sprintf("aws:elasticache:%s:%s", region, nodeType)
	
	if cached, ok := p.cache.Get(cacheKey); ok {
		return cached.(*DatabasePricing), nil
	}

	// Static pricing for MVP
	basePrices := map[string]float64{
		"cache.t3.micro":  0.017,
		"cache.t3.small":  0.034,
		"cache.t3.medium": 0.068,
		"cache.m5.large":  0.166,
		"cache.m5.xlarge": 0.332,
		"cache.r5.large":  0.228,
		"cache.r5.xlarge": 0.456,
	}

	hourlyRate, exists := basePrices[nodeType]
	if !exists {
		hourlyRate = 0.1
	}

	monthlyCost := hourlyRate * 24 * 30

	pricing := &DatabasePricing{
		InstanceType: nodeType,
		Engine:       "redis",
		MonthlyCost:  monthlyCost,
		BackupCost:   0, // ElastiCache backups are free up to storage size
		Provider:     "aws",
	}

	p.cache.Set(cacheKey, pricing, 24*time.Hour)
	return pricing, nil
}

// CalculateDataTransferCost calculates network egress costs
func (p *PricingClient) CalculateDataTransferCost(ctx context.Context, provider CloudProvider, region string, transferOutGB float64, crossAZ bool) (*NetworkPricing, error) {
	cacheKey := fmt.Sprintf("%s:network:%s:%f:%v", provider, region, transferOutGB, crossAZ)
	
	if cached, ok := p.cache.Get(cacheKey); ok {
		return cached.(*NetworkPricing), nil
	}

	var costPerGB float64
	
	switch provider {
	case ProviderAWS:
		// AWS pricing (us-east-1 as baseline)
		if crossAZ {
			costPerGB = 0.01 // $0.01/GB cross-AZ
		} else {
			// Tiered internet egress
			if transferOutGB <= 100*1024 { // First 100TB
				costPerGB = 0.09
			} else if transferOutGB <= 500*1024 { // Next 400TB
				costPerGB = 0.085
			} else { // Over 500TB
				costPerGB = 0.07
			}
		}
		
	case ProviderGCP:
		if crossAZ {
			costPerGB = 0.01 // Same as AWS
		} else {
			costPerGB = 0.085 // Slightly cheaper than AWS
		}
		
	case ProviderAzure:
		if crossAZ {
			costPerGB = 0.01
		} else {
			costPerGB = 0.087
		}
	}

	totalCost := transferOutGB * costPerGB

	pricing := &NetworkPricing{
		DataTransferOutGB: transferOutGB,
		CostPerGB:         costPerGB,
		TotalCost:         totalCost,
		CrossAZ:           crossAZ,
		Provider:          string(provider),
	}

	p.cache.Set(cacheKey, pricing, 24*time.Hour)
	return pricing, nil
}

// GetS3Pricing calculates S3 storage costs
func (p *PricingClient) GetS3Pricing(ctx context.Context, region string, storageGB float64, requests int64) (float64, error) {
	cacheKey := fmt.Sprintf("aws:s3:%s:%f:%d", region, storageGB, requests)
	
	if cached, ok := p.cache.Get(cacheKey); ok {
		return cached.(float64), nil
	}

	// S3 Standard pricing (us-east-1)
	var storageCost float64
	
	// Tiered storage pricing
	if storageGB <= 50*1024 { // First 50TB
		storageCost = storageGB * 0.023
	} else if storageGB <= 500*1024 { // Next 450TB
		storageCost = (50 * 1024 * 0.023) + ((storageGB - 50*1024) * 0.022)
	} else { // Over 500TB
		storageCost = (50 * 1024 * 0.023) + (450 * 1024 * 0.022) + ((storageGB - 500*1024) * 0.021)
	}

	// Request pricing (per 1000 requests)
	requestCost := float64(requests) / 1000.0 * 0.005 // $0.005 per 1K GET requests

	totalCost := storageCost + requestCost

	p.cache.Set(cacheKey, totalCost, 24*time.Hour)
	return totalCost, nil
}

// MemoryCache for pricing data
type MemoryCache struct {
	data map[string]cacheEntry
	ttl  time.Duration
}

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

func NewMemoryCache(ttl time.Duration) *MemoryCache {
	return &MemoryCache{
		data: make(map[string]cacheEntry),
		ttl:  ttl,
	}
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	entry, ok := c.data[key]
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			delete(c.data, key)
		}
		return nil, false
	}
	return entry.value, true
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// CompareProviders compares costs across AWS, GCP, Azure
func (p *PricingClient) CompareProviders(ctx context.Context, region string, instanceType string) (map[string]InstancePricing, error) {
	// For MVP, return AWS only with note about multi-cloud comparison
	// In production, implement GCP and Azure pricing APIs
	
	awsPricing, err := p.GetEC2Pricing(ctx, region, []string{instanceType})
	if err != nil {
		return nil, err
	}

	result := make(map[string]InstancePricing)
	for _, pricing := range awsPricing {
		result["aws"] = pricing
		// TODO: Add GCP and Azure equivalents
	}

	return result, nil
}
