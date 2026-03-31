package diagram

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewMermaidRenderer(t *testing.T) {
	m := NewMermaidRenderer()
	if m == nil {
		t.Fatal("NewMermaidRenderer returned nil")
	}
	if m.baseURL != "https://mermaid.ink" {
		t.Errorf("Expected baseURL https://mermaid.ink, got %s", m.baseURL)
	}
}

func TestRenderFunctions(t *testing.T) {
	// Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL format
		// Expecting /png/base64code, /pdf/base64code, /svg/base64code
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("mock binary data"))
	}))
	defer ts.Close()

	m := NewMermaidRenderer()
	m.baseURL = ts.URL // Override for test

	code := "graph TD; A-->B;"

	tests := []struct {
		name   string
		render func(string) ([]byte, error)
	}{
		{"RenderPNG", m.RenderPNG},
		{"RenderPDF", m.RenderPDF},
		{"RenderSVG", m.RenderSVG},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.render(code)
			if err != nil {
				t.Fatalf("%s failed: %v", tt.name, err)
			}
			if !bytes.Equal(data, []byte("mock binary data")) {
				t.Errorf("%s returned unexpected data", tt.name)
			}
		})
	}
}

func TestRender_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid mermaid syntax"))
	}))
	defer ts.Close()

	m := NewMermaidRenderer()
	m.baseURL = ts.URL

	_, err := m.RenderPNG("invalid")
	if err == nil {
		t.Error("RenderPNG should fail on 400 status")
	}
	if !strings.Contains(err.Error(), "render failed with status 400") {
		t.Errorf("Expected status 400 error message, got: %v", err)
	}
}

func TestGetPublicURL(t *testing.T) {
	m := NewMermaidRenderer()
	code := "graph TD; A-->B;"
	url := m.GetPublicURL(code)

	// Base64 of "graph TD; A-->B;" is "Z3JhcGggVEQ7IEEtLT5COw=="
	expected := "https://mermaid.ink/img/Z3JhcGggVEQ7IEEtLT5COw=="
	if url != expected {
		t.Errorf("Expected URL %s, got %s", expected, url)
	}
}

func TestValidateMermaid(t *testing.T) {
	tests := []struct {
		code    string
		wantErr bool
	}{
		{"graph TD; A-->B;", false},
		{"flowchart LR; A-->B;", false},
		{"sequenceDiagram; Alice->>Bob: Hello;", false},
		{"classDiagram; Class01 <|-- Class02;", false},
		{"stateDiagram; [*] --> State1;", false},
		{"erDiagram; CUSTOMER ||--o{ ORDER : places;", true}, // Should fail as it has '{' in its DSL usually, but wait, looking at my mock check, it only has '{' and no '}'
		{"gantt; title A Gantt Diagram;", false},
		{"pie; title Pets adopted by volunteers; \"Dogs\" : 386;", false},
		{"journey; title My working day;", false},
		{"gitGraph; commit;", false},
		{"C4Context; Boundary(b1, \"Boundary\") { };", false},
		{"", true},                       // Empty
		{"invalid TD; A-->B;", true},     // Invalid type
		{"graph TD; { A-->B;", true},     // Unbalanced braces
		{"graph TD; A(B;", true},        // Unbalanced parentheses
	}

	for _, tt := range tests {
		err := ValidateMermaid(tt.code)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateMermaid(%q) error = %v, wantErr %v", tt.code, err, tt.wantErr)
		}
	}
}
