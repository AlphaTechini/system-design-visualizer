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

// DesignHandler handles design creation and retrieval
type DesignHandler struct {
	db            *database.SupabaseClient
	aiProvider    *ai.NEARAIProvider
	costEstimator *cost.CostEstimator
	tfGenerator   *terraform.Generator
	mermaidRender *diagram.MermaidRenderer
}

// NewDesignHandler creates handler with all dependencies
func NewDesignHandler(
	db *database.SupabaseClient,
	nearAPIKey string,
	gcpAPIKey string,
) *DesignHandler {
	return &DesignHandler{
		db:            db,
		aiProvider:    ai.NewNEARAIProvider(nearAPIKey, "deepseek-ai/DeepSeek-V3.1"),
		costEstimator: cost.NewCostEstimator(gcpAPIKey),
		tfGenerator:   terraform.NewGenerator(),
		mermaidRender: diagram.NewMermaidRenderer(),
	}
}

// CreateDesignRequest represents incoming design request
type CreateDesignRequest struct {
	Requirements     map[string]interface{} `json:"requirements"`
	Clarifications   string                 `json:"clarifications,omitempty"`
	PreferredCloud   string                 `json:"preferred_cloud,omitempty"` // aws, gcp, azure
	IncludeTerraform bool                   `json:"include_terraform,omitempty"`
	IncludeDiagrams  bool                   `json:"include_diagrams,omitempty"`
}

// CreateDesignResponse represents design creation response
type CreateDesignResponse struct {
	DesignID        string                     `json:"design_id"`
	Status          string                     `json:"status"`
	Architecture    string                     `json:"architecture,omitempty"`
	CostEstimate    *cost.ArchitectureCost     `json:"cost_estimate,omitempty"`
	TerraformFiles  map[string]string          `json:"terraform_files,omitempty"`
	DiagramURL      string                     `json:"diagram_url,omitempty"`
	CaseStudies     []string                   `json:"case_studies,omitempty"`
	Recommendations []string                   `json:"recommendations,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
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

	// Step 2: Extract architecture spec from AI response
	spec := h.extractArchitectureSpec(architecture, req.PreferredCloud)

	// Step 3: Generate cost estimate
	costEstimate, err := h.costEstimator.EstimateArchitecture(ctx, spec.Provider, spec.Region, spec)
	if err != nil {
		http.Error(w, fmt.Sprintf("cost estimation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Step 4: Generate Terraform (if requested)
	var tfFiles map[string]string
	if req.IncludeTerraform {
		tfFiles, err = h.tfGenerator.Generate(spec)
		if err != nil {
			http.Error(w, fmt.Sprintf("Terraform generation failed: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Step 5: Generate Mermaid diagram (if requested)
	var diagramURL string
	if req.IncludeDiagrams {
		diagramPrompt := ai.PromptDiagramDescription(architecture)
		mermaidCode, err := h.aiProvider.Chat(ctx, ai.SystemPromptArchitect, diagramPrompt)
		if err != nil {
			http.Error(w, fmt.Sprintf("diagram generation failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Validate mermaid code
		if err := diagram.ValidateMermaid(mermaidCode); err != nil {
			http.Error(w, fmt.Sprintf("invalid mermaid code: %v", err), http.StatusBadRequest)
			return
		}

		// Render to PNG
		pngBytes, err := h.mermaidRender.RenderPNG(mermaidCode)
		if err != nil {
			http.Error(w, fmt.Sprintf("diagram render failed: %v", err), http.StatusInternalServerError)
			return
		}

		// TODO: Upload PNG to storage and get URL
		// For now, use public mermaid.ink URL
		diagramURL = h.mermaidRender.GetPublicURL(mermaidCode)
	}

	// Step 6: Find relevant case studies via Exa (if configured)
	// TODO: Integrate Exa for case study lookup

	// Step 7: Save design to database
	err = h.saveDesign(ctx, designID, req, architecture, costEstimate, tfFiles, diagramURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("save design failed: %v", err), http.StatusInternalServerError)
		return
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

// extractArchitectureSpec converts AI text response to structured spec
func (h *DesignHandler) extractArchitectureSpec(architecture string, preferredCloud string) cost.ArchitectureSpec {
	// Default values
	spec := cost.ArchitectureSpec{
		Provider:        "aws",
		Region:          "us-east-1",
		InstanceType:    "t3.micro",
		InstanceCount:   2,
		DatabaseType:    "postgres",
		DatabaseSize:    "db.t3.micro",
		DatabaseStorageGB: 20,
		CacheType:       "cache.t3.micro",
		CacheNodes:      1,
		DataTransferOutGB: 100,
		StorageGB:       10,
		StorageRequests: 10000,
		CrossAZ:         false,
		ExpectedUsers:   10000,
		HealthCheckPath: "/health",
	}

	// Override with preferred cloud
	if preferredCloud != "" {
		spec.Provider = preferredCloud
	}

	// TODO: Use AI to parse architecture text and extract actual values
	// For MVP, using sensible defaults

	return spec
}

// saveDesign persists design to database
func (h *DesignHandler) saveDesign(
	ctx context.Context,
	designID string,
	req CreateDesignRequest,
	architecture string,
	costEstimate *cost.ArchitectureCost,
	tfFiles map[string]string,
	diagramURL string,
) error {
	// Serialize terraform files to JSON
	var tfJSON []byte
	if tfFiles != nil {
		tfJSON, _ = json.Marshal(tfFiles)
	}

	// Serialize cost estimate to JSON
	costJSON, _ := json.Marshal(costEstimate)

	// Insert into database
	_, err := h.db.Pool().Exec(ctx, `
		INSERT INTO designs (
			id, requirements_json, ai_recommendations_json,
			mermaid_code, terraform_code, cost_estimate_json,
			status
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, designID, toJSON(req.Requirements), toJSON(map[string]string{
		"architecture": architecture,
	}), diagramURL, string(tfJSON), string(costJSON), "completed")

	return err
}

// Helper: Convert interface to JSON string
func toJSON(v interface{}) string {
	if v == nil {
		return "{}"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// GetDesign handles GET /api/v1/designs/{id}
func (h *DesignHandler) GetDesign(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement design retrieval
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "not_implemented",
	})
}

// ListDesigns handles GET /api/v1/designs
func (h *DesignHandler) ListDesigns(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement design listing
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"designs": []interface{}{},
		"total":   0,
	})
}

// RegenerateTerraform handles POST /api/v1/designs/{id}/terraform
func (h *DesignHandler) RegenerateTerraform(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement terraform regeneration with updated specs
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "not_implemented",
	})
}

// CompareProviders handles POST /api/v1/designs/{id}/compare
func (h *DesignHandler) CompareProviders(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement multi-cloud cost comparison
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "not_implemented",
	})
}
