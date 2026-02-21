package diagram

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MermaidRenderer converts Mermaid code to PNG/PDF/SVG
type MermaidRenderer struct {
	baseURL    string
	httpClient *http.Client
}

// RenderFormat specifies output format
type RenderFormat string

const (
	FormatPNG  RenderFormat = "png"
	FormatPDF  RenderFormat = "pdf"
	FormatSVG  RenderFormat = "svg"
)

// NewMermaidRenderer creates renderer with Mermaid.ink
func NewMermaidRenderer() *MermaidRenderer {
	return &MermaidRenderer{
		baseURL: "https://mermaid.ink",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RenderPNG converts Mermaid code to PNG bytes
func (m *MermaidRenderer) RenderPNG(mermaidCode string) ([]byte, error) {
	return m.render(mermaidCode, FormatPNG)
}

// RenderPDF converts Mermaid code to PDF bytes
func (m *MermaidRenderer) RenderPDF(mermaidCode string) ([]byte, error) {
	return m.render(mermaidCode, FormatPDF)
}

// RenderSVG converts Mermaid code to SVG bytes
func (m *MermaidRenderer) RenderSVG(mermaidCode string) ([]byte, error) {
	return m.render(mermaidCode, FormatSVG)
}

// render is the internal implementation
func (m *MermaidRenderer) render(mermaidCode string, format RenderFormat) ([]byte, error) {
	// Encode mermaid code as base64
	encoded := base64.StdEncoding.EncodeToString([]byte(mermaidCode))

	// URL format: https://mermaid.ink/{format}/{base64_code}
	url := fmt.Sprintf("%s/%s/%s", m.baseURL, format, encoded)

	resp, err := m.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("mermaid.ink request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("render failed with status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// GetPublicURL returns public URL for sharing (no download)
func (m *MermaidRenderer) GetPublicURL(mermaidCode string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(mermaidCode))
	return fmt.Sprintf("%s/img/%s", m.baseURL, encoded)
}

// ValidateMermaid checks basic syntax (client-side validation)
func ValidateMermaid(code string) error {
	// Basic validation rules
	if len(code) == 0 {
		return fmt.Errorf("empty mermaid code")
	}

	// Check for required diagram type declaration
	validTypes := []string{
		"graph ",
		"flowchart ",
		"sequenceDiagram",
		"classDiagram",
		"stateDiagram",
		"erDiagram",
		"gantt",
		"pie",
		"journey",
		"gitGraph",
		"C4Context",
	}

	hasValidType := false
	for _, validType := range validTypes {
		if contains(code, validType) {
			hasValidType = true
			break
		}
	}

	if !hasValidType {
		return fmt.Errorf("invalid diagram type - must start with: graph, flowchart, sequenceDiagram, etc.")
	}

	// Check for balanced braces
	openBraces := countChar(code, '{')
	closeBraces := countChar(code, '}')
	if openBraces != closeBraces {
		return fmt.Errorf("unbalanced braces: %d open, %d close", openBraces, closeBraces)
	}

	// Check for balanced parentheses
	openParens := countChar(code, '(')
	closeParens := countChar(code, ')')
	if openParens != closeParens {
		return fmt.Errorf("unbalanced parentheses: %d open, %d close", openParens, closeParens)
	}

	return nil
}

// Helper functions
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func countChar(s string, c byte) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			count++
		}
	}
	return count
}
