package cost

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// AWSPricingClient fetches real AWS pricing via Price List API
type AWSPricingClient struct {
	httpClient *http.Client
	cache      *MemoryCache
}

// NewAWSPricingClient creates AWS pricing client
func NewAWSPricingClient() *AWSPricingClient {
	return &AWSPricingClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cache:      NewMemoryCache(6 * time.Hour), // Cache for 6 hours
	}
}

// GetProducts queries AWS Price List API
func (a *AWSPricingClient) GetProducts(ctx context.Context, serviceCode string, filters map[string]string) ([]AWSProduct, error) {
	cacheKey := fmt.Sprintf("aws:products:%s:%v", serviceCode, filters)
	
	if cached, ok := a.cache.Get(cacheKey); ok {
		return cached.([]AWSProduct), nil
	}

	// AWS Price List API endpoint
	baseURL := "https://pricing.us-east-1.amazonaws.com/products"
	
	params := url.Values{}
	params.Set("serviceCode", serviceCode)
	
	// Build filter query
	for key, value := range filters {
		params.Add(key, value)
	}

	reqURL := baseURL + "?" + params.Encode()
	
	resp, err := a.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("AWS Price List API error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var awsResp AWSPriceListResponse
	if err := json.Unmarshal(body, &awsResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	a.cache.Set(cacheKey, awsResp.Products, 6*time.Hour)
	return awsResp.Products, nil
}

// AWSPriceListResponse matches AWS Price List API response
type AWSPriceListResponse struct {
	Products       []AWSProduct `json:"products"`
	PublicationID  string       `json:"publicationId"`
	NextPageToken  string       `json:"nextPageToken,omitempty"`
	FormatVersion  string       `json:"formatVersion"`
}

// AWSProduct represents an AWS product/SKU
type AWSProduct struct {
	SKU           string                 `json:"sku"`
	ProductFamily string                 `json:"productFamily"`
	Attributes    AWSProductAttributes   `json:"attributes"`
	Terms         map[string]interface{} `json:"terms"`
}

// AWSProductAttributes contains product metadata
type AWSProductAttributes struct {
	InstanceType     string `json:"instanceType"`
	Memory           string `json:"memory"`
	VCPU             string `json:"vcpu"`
	OperatingSystem  string `json:"operatingSystem"`
	Location         string `json:"location"`
	Tenancy          string `json:"tenancy"`
	CapacityStatus   string `json:"capacityStatus"`
	CurrentGeneration string `json:"currentGeneration"`
}

// GCPPricingClient fetches real GCP pricing via Cloud Billing Pricing API
type GCPPricingClient struct {
	httpClient *http.Client
	apiKey     string
	cache      *MemoryCache
}

// NewGCPPricingClient creates GCP pricing client
func NewGCPPricingClient(apiKey string) *GCPPricingClient {
	return &GCPPricingClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     apiKey,
		cache:      NewMemoryCache(6 * time.Hour),
	}
}

// GetServices lists all GCP services
func (g *GCPPricingClient) GetServices(ctx context.Context) ([]GCPService, error) {
	cacheKey := "gcp:services"
	
	if cached, ok := g.cache.Get(cacheKey); ok {
		return cached.([]GCPService), nil
	}

	baseURL := "https://cloudbilling.googleapis.com/v2beta/services"
	reqURL := fmt.Sprintf("%s?key=%s&pageSize=500", baseURL, g.apiKey)

	resp, err := g.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("GCP Pricing API error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var gcpResp GCPServiceListResponse
	if err := json.Unmarshal(body, &gcpResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	g.cache.Set(cacheKey, gcpResp.Services, 6*time.Hour)
	return gcpResp.Services, nil
}

// GetSKUs gets SKUs for a specific service
func (g *GCPPricingClient) GetSKUs(ctx context.Context, serviceName string) ([]GCPSKU, error) {
	cacheKey := fmt.Sprintf("gcp:skus:%s", serviceName)
	
	if cached, ok := g.cache.Get(cacheKey); ok {
		return cached.([]GCPSKU), nil
	}

	baseURL := "https://cloudbilling.googleapis.com/v2beta/skus"
	filter := fmt.Sprintf("service=\"%s\"", serviceName)
	reqURL := fmt.Sprintf("%s?key=%s&filter=%s&pageSize=5000", baseURL, g.apiKey, url.QueryEscape(filter))

	resp, err := g.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("GCP SKU API error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var gcpResp GCPSKUListResponse
	if err := json.Unmarshal(body, &gcpResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	g.cache.Set(cacheKey, gcpResp.Skus, 6*time.Hour)
	return gcpResp.Skus, nil
}

// GCPServiceListResponse matches GCP Services API response
type GCPServiceListResponse struct {
	Services      []GCPService `json:"services"`
	NextPageToken string       `json:"nextPageToken,omitempty"`
}

// GCPService represents a GCP service
type GCPService struct {
	Name        string `json:"name"`
	ServiceID   string `json:"serviceId"`
	DisplayName string `json:"displayName"`
}

// GCPSKUListResponse matches GCP SKUs API response
type GCPSKUListResponse struct {
	Skus          []GCPSKU `json:"skus"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

// GCPSKU represents a GCP SKU with pricing
type GCPSKU struct {
	SKUID                  string           `json:"skuId"`
	ServiceName            string           `json:"serviceName"`
	Description            string           `json:"description"`
	Category               GKPSKUCategory   `json:"category"`
	PricingInfo            []GCPPricingInfo `json:"pricingInfo"`
	ServiceRegion          string           `json:"serviceRegion"`
	PricingUnit            string           `json:"pricingUnit"`
	SupportedUsageQuantity string           `json:"supportedUsageQuantity"`
}

// GKPSKUCategory contains SKU categorization
type GKPSKUCategory struct {
	ResourceFamily string `json:"resourceFamily"`
	ResourceGroup  string `json:"resourceGroup"`
	UsageType      string `json:"usageType"`
	ServiceType    string `json:"serviceType"`
}

// GCPPricingInfo contains pricing details
type GCPPricingInfo struct {
	Summary               string             `json:"summary"`
	PricingExpression     GCPPricingExpression `json:"pricingExpression"`
	CurrencyConversionRate float64            `json:"currencyConversionRate"`
	EffectiveTime         string             `json:"effectiveTime"`
}

// GCPPricingExpression contains the actual pricing formula
type GCPPricingExpression struct {
	UsageUnit                string         `json:"usageUnit"`
	DisplayQuantity          int            `json:"displayQuantity"`
	UsageUnitDescription     string         `json:"usageUnitDescription"`
	BaseUnit                 string         `json:"baseUnit"`
	BaseUnitConversionFactor int64          `json:"baseUnitConversionFactor"`
	TieredRates              []GCPTieredRate `json:"tieredRates"`
}

// GCPTieredRate contains tiered pricing
type GCPTieredRate struct {
	StartUsageAmount int64            `json:"startUsageAmount"`
	UnitPrice        GCPUnitPrice     `json:"unitPrice"`
}

// GCPUnitPrice contains unit pricing
type GCPUnitPrice struct {
	CurrencyCode string  `json:"currencyCode"`
	Units        string  `json:"units"`
	Nanos        float64 `json:"nanos"`
}

// AzurePricingClient fetches real Azure pricing via Retail Prices API
type AzurePricingClient struct {
	httpClient *http.Client
	cache      *MemoryCache
}

// NewAzurePricingClient creates Azure pricing client
func NewAzurePricingClient() *AzurePricingClient {
	return &AzurePricingClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cache:      NewMemoryCache(6 * time.Hour),
	}
}

// GetRetailPrices queries Azure Retail Prices API
func (a *AzurePricingClient) GetRetailPrices(ctx context.Context, filters map[string]string) ([]AzureRetailPrice, error) {
	cacheKey := fmt.Sprintf("azure:prices:%v", filters)
	
	if cached, ok := a.cache.Get(cacheKey); ok {
		return cached.([]AzureRetailPrice), nil
	}

	baseURL := "https://prices.azure.com/api/retail/prices"
	
	params := url.Values{}
	params.Set("$top", "1000")
	
	// Build OData filter
	if len(filters) > 0 {
		filterParts := []string{}
		for key, value := range filters {
			filterParts = append(filterParts, fmt.Sprintf("%s eq '%s'", key, value))
		}
		if len(filterParts) > 0 {
			params.Set("$filter", joinWithAnd(filterParts))
		}
	}

	reqURL := baseURL + "?" + params.Encode()
	
	resp, err := a.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("Azure Retail Prices API error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var azureResp AzureRetailPricesResponse
	if err := json.Unmarshal(body, &azureResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	a.cache.Set(cacheKey, azureResp.Items, 6*time.Hour)
	return azureResp.Items, nil
}

// AzureRetailPricesResponse matches Azure Retail Prices API response
type AzureRetailPricesResponse struct {
	BillingCurrency    string                 `json:"BillingCurrency"`
	CurrencyCode       string                 `json:"CurrencyCode"`
	CustomerEntityID   string                 `json:"CustomerEntityId"`
	CustomerEntityType string                 `json:"CustomerEntityType"`
	Items              []AzureRetailPrice     `json:"Items"`
	NextPageLink       string                 `json:"NextPageLink"`
	Count              int                    `json:"Count"`
}

// AzureRetailPrice represents a single price item
type AzureRetailPrice struct {
	CurrencyCode         string  `json:"currencyCode"`
	TierMinimumUnits     float64 `json:"tierMinimumUnits"`
	RetailPrice          float64 `json:"retailPrice"`
	UnitPrice            float64 `json:"unitPrice"`
	ArmRegionName        string  `json:"armRegionName"`
	Location             string  `json:"location"`
	EffectiveStartDate   string  `json:"effectiveStartDate"`
	MeterID              string  `json:"meterId"`
	MeterName            string  `json:"meterName"`
	ProductID            string  `json:"productId"`
	SkuID                string  `json:"skuId"`
	ProductName          string  `json:"productName"`
	SkuName              string  `json:"skuName"`
	ServiceName          string  `json:"serviceName"`
	ServiceID            string  `json:"serviceId"`
	ServiceFamily        string  `json:"serviceFamily"`
	UnitOfMeasure        string  `json:"unitOfMeasure"`
	Type                 string  `json:"type"`
	IsPrimaryMeterRegion bool    `json:"isPrimaryMeterRegion"`
	ArmSkuName           string  `json:"armSkuName"`
	ReservationTerm      string  `json:"reservationTerm,omitempty"`
	EffectiveEndDate     string  `json:"effectiveEndDate,omitempty"`
}

// Helper function
func joinWithAnd(parts []string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " and "
		}
		result += part
	}
	return result
}

// GetRDSPricing fetches AWS RDS pricing (stub - uses static pricing for MVP)
func (a *AWSPricingClient) GetRDSPricing(ctx context.Context, region string, engine string, instanceType string) (*DatabasePricing, error) {
	// TODO: Integrate with AWS Price List API
	// For now, return static pricing
	basePrices := map[string]float64{
		"db.t3.micro":  0.017,
		"db.t3.small":  0.034,
		"db.m5.large":  0.171,
	}
	
	hourlyRate, exists := basePrices[instanceType]
	if !exists {
		hourlyRate = 0.1
	}
	
	engineMultiplier := 1.0
	if engine == "oracle-ee" || engine == "sqlserver-ee" {
		engineMultiplier = 3.0
	}
	
	monthlyCost := hourlyRate * engineMultiplier * 24 * 30
	
	return &DatabasePricing{
		InstanceType: instanceType,
		Engine:       engine,
		MonthlyCost:  monthlyCost,
		BackupCost:   monthlyCost * 0.2,
		Provider:     "aws",
	}, nil
}

// GetElastiCachePricing fetches AWS ElastiCache pricing (stub)
func (a *AWSPricingClient) GetElastiCachePricing(ctx context.Context, region string, nodeType string) (*DatabasePricing, error) {
	basePrices := map[string]float64{
		"cache.t3.micro": 0.017,
		"cache.t3.small": 0.034,
		"cache.m5.large": 0.166,
	}
	
	hourlyRate, exists := basePrices[nodeType]
	if !exists {
		hourlyRate = 0.1
	}
	
	monthlyCost := hourlyRate * 24 * 30
	
	return &DatabasePricing{
		InstanceType: nodeType,
		Engine:       "redis",
		MonthlyCost:  monthlyCost,
		Provider:     "aws",
	}, nil
}

// CalculateDataTransferCost calculates AWS data transfer costs (stub)
func (a *AWSPricingClient) CalculateDataTransferCost(ctx context.Context, provider CloudProvider, region string, transferOutGB float64, crossAZ bool) (*NetworkPricing, error) {
	costPerGB := 0.09
	if crossAZ {
		costPerGB = 0.01
	}
	
	return &NetworkPricing{
		DataTransferOutGB: transferOutGB,
		CostPerGB:         costPerGB,
		TotalCost:         transferOutGB * costPerGB,
		CrossAZ:           crossAZ,
		Provider:          "aws",
	}, nil
}

// GetS3Pricing calculates AWS S3 storage costs (stub)
func (a *AWSPricingClient) GetS3Pricing(ctx context.Context, region string, storageGB float64, requests int64) (float64, error) {
	storageCost := storageGB * 0.023 // $0.023/GB for first 50TB
	requestCost := float64(requests) / 1000.0 * 0.005 // $0.005 per 1K requests
	
	return storageCost + requestCost, nil
}

// DatabasePricing represents managed database costs
type DatabasePricing struct {
	InstanceType string  `json:"instance_type"`
	Engine       string  `json:"engine"`
	MonthlyCost  float64 `json:"monthly_cost"`
	BackupCost   float64 `json:"backup_cost"`
	Provider     string  `json:"provider"`
}

// NetworkPricing represents data transfer costs
type NetworkPricing struct {
	DataTransferOutGB float64 `json:"data_transfer_out_gb"`
	CostPerGB         float64 `json:"cost_per_gb"`
	TotalCost         float64 `json:"total_cost"`
	CrossAZ           bool    `json:"cross_az"`
	Provider          string  `json:"provider"`
}
