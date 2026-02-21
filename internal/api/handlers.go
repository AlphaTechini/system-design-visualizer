package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AlphaTechini/system-design-visualizer/internal/ai"
	"github.com/AlphaTechini/system-design-visualizer/internal/cost"
	"github.com/AlphaTechini/system-design-visualizer/internal/database"
	"github.com/AlphaTechini/system-design-visualizer/internal/diagram"
	"github.com/AlphaTechini/system-design-visualizer/internal/terraform"
	"github.com/google/uuid"
)

// DesignHandler handles design creation
type DesignHandler struct {
	db            *database.SupabaseClient
	aiProvider    *ai.NEARAIProvider
	costEstimator *cost.CostEstimator
	tfGenerator   *terraform.Generator
	mermaidRender *diagram.MermaidRenderer
}

// NewDesignHandler creates handler
func NewDesignHandler(db *database.SupabaseClient, nearAPIKey string, gcpAPIKey string) *DesignHandler {
	return &DesignHandler{
		db:            db,
		aiProvider:    ai.NewNEARAIProvider(nearAPIKey, "deepseek-ai/DeepSeek-V3.1"),
		costEstimator: cost.NewCostEstimator(gcpAPIKey),
		tfGenerator:   terraform.NewGenerator(),
		mermaidRender: diagram.NewMermaidRenderer(),
	}
}

// CreateDesignRequest represents incoming request
type CreateDesignRequest struct {
	Requirements     map[string]interface{} `json:"requirements"`
	Clarifications   string                 `json:"clarifications,omitempty"`
	PreferredCloud   string                 `json:"preferred_cloud,omitempty"`
	IncludeTerraform bool                   `json:"include_terraform,omitempty"`
	IncludeDiagrams  bool                   `json:"include_diagrams,omitempty"`
}

// CreateDesignResponse represents response
type CreateDesignResponse struct {
	DesignID       string                 `json:"design_id"`
	Status         string                 `json:"status"`
	Architecture   string                 `json:"architecture,omitempty"`
	CostEstimate   *cost.ArchitectureCost `json:"cost_estimate,omitempty"`
	TerraformFiles map[string]string      `json:"terraform_files,omitempty"`
	DiagramURL     string                 `json:"diagram_url,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// CreateDesign handles POST /api/v1/designs
func (h *DesignHandler) CreateDesign(w http.ResponseWriter, r *http.Request) {
	var req CreateDesignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("decode request: %v", err), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	designID := uuid.New().String()

	// Step 1: Get AI architecture recommendation
	archPrompt := ai.PromptArchitectureRecommendation(req.Requirements, req.Clarifications)
	architecture, err := h.aiProvider.Chat(ctx, ai.SystemPromptArchitect, archPrompt)
	if err != nil {
		http.Error(w, fmt.Sprintf("AI recommendation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Step 2: Create simple spec for cost estimation
	spec := cost.ArchitectureSpec{
		Provider:        "aws",
		Region:          "us-east-1",
		InstanceCount:   2,
		ExpectedUsers:   10000,
		DataTransferOutGB: 100,
		StorageGB:       10,
	}
	
	if req.PreferredCloud != "" {
		spec.Provider = req.PreferredCloud
	}

	// Step 3: Generate cost estimate
	provider := cost.CloudProvider(spec.Provider)
	costEstimate, err := h.costEstimator.EstimateArchitecture(ctx, provider, spec.Region, spec)
	if err != nil {
		http.Error(w, fmt.Sprintf("cost estimation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Step 4: Generate Terraform (if requested)
	var tfFiles map[string]string
	if req.IncludeTerraform {
		tfSpec := terraform.ArchitectureSpec{
			Provider:    spec.Provider,
			Region:      spec.Region,
			ProjectName: "project",
		}
		tfFiles, err = h.tfGenerator.Generate(tfSpec)
		if err != nil {
			http.Error(w, fmt.Sprintf("Terraform generation failed: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Step 5: Generate diagram URL (if requested)
	var diagramURL string
	if req.IncludeDiagrams {
		diagramURL = "https://mermaid.ink/img/example" // Placeholder
	}

	// Build response
	response := CreateDesignResponse{
		DesignID:       designID,
		Status:         "completed",
		Architecture:   architecture,
		CostEstimate:   costEstimate,
		TerraformFiles: tfFiles,
		DiagramURL:     diagramURL,
		CreatedAt:      time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// ListDesigns handles GET /api/v1/designs
func (h *DesignHandler) ListDesigns(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"designs": []interface{}{},
		"total":   0,
	})
}

// GetDesign handles GET /api/v1/designs/{id}
func (h *DesignHandler) GetDesign(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "not_implemented",
	})
}

// RegenerateTerraform handles POST /api/v1/designs/{id}/terraform
func (h *DesignHandler) RegenerateTerraform(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "not_implemented",
	})
}

// CompareProviders handles POST /api/v1/designs/{id}/compare
func (h *DesignHandler) CompareProviders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "not_implemented",
	})
}
