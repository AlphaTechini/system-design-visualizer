package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewNEARAIProvider(t *testing.T) {
	n := NewNEARAIProvider("key", "model")
	if n.apiKey != "key" {
		t.Errorf("Expected apiKey key, got %s", n.apiKey)
	}
	if n.model != "model" {
		t.Errorf("Expected model model, got %s", n.model)
	}

	n2 := NewNEARAIProvider("key", "")
	if n2.model != "deepseek-ai/DeepSeek-V3.1" {
		t.Errorf("Expected default model, got %s", n2.model)
	}
}

func TestChat_Success(t *testing.T) {
	// Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected test-key, got %s", r.Header.Get("Authorization"))
		}

		resp := ChatCompletionResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "test-model",
			Choices: []struct {
				Index        int     `json:"index"`
				Message      Message `json:"message"`
				FinishReason string  `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: Message{
						Role:    "assistant",
						Content: "mock architecture recommendation",
					},
					FinishReason: "stop",
				},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	n := NewNEARAIProvider("test-key", "test-model")
	n.baseURL = ts.URL // Override for test

	// Clear cache for clean test
	memoryCache.data = make(map[string]cacheEntry)

	ans, err := n.Chat(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if ans != "mock architecture recommendation" {
		t.Errorf("Expected mock architecture recommendation, got %s", ans)
	}

	// Test caching
	// Second call should not hit mock server (if it did and we counted, but here we just check it returns the same)
	// We can modify mock server to fail on second call to ensure cache hit.
	ts.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Mock server should not have been called on cached request")
	})

	ans2, err := n.Chat(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("Cached Chat failed: %v", err)
	}
	if ans2 != "mock architecture recommendation" {
		t.Errorf("Expected cached mock architecture recommendation, got %s", ans2)
	}
}

func TestChat_ErrorScenarios(t *testing.T) {
	n := NewNEARAIProvider("test-key", "test-model")

	// 1. API error status
	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer ts1.Close()

	n.baseURL = ts1.URL
	_, err := n.Chat(context.Background(), "s", "u")
	if err == nil || !strings.Contains(err.Error(), "API error 401") {
		t.Errorf("Expected 401 error, got %v", err)
	}

	// 2. Empty choices
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatCompletionResponse{
			Choices: []struct {
				Index        int     `json:"index"`
				Message      Message `json:"message"`
				FinishReason string  `json:"finish_reason"`
			}{},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts2.Close()

	memoryCache.data = make(map[string]cacheEntry) // reset cache
	n.baseURL = ts2.URL
	_, err = n.Chat(context.Background(), "s2", "u2")
	if err == nil || err.Error() != "no choices in response" {
		t.Errorf("Expected no choices error, got %v", err)
	}
}

func TestPromptGenerators(t *testing.T) {
	reqs := map[string]interface{}{"users": 1000}

	if q := PromptClarifyingQuestions(reqs); !strings.Contains(q, "users: 1000") {
		t.Error("PromptClarifyingQuestions missing requirement")
	}

	if r := PromptArchitectureRecommendation(reqs, "clarified"); !strings.Contains(r, "clarified") {
		t.Error("PromptArchitectureRecommendation missing clarification")
	}

	if c := PromptCostEstimation("arch", "aws"); !strings.Contains(c, "arch") || !strings.Contains(c, "aws") {
		t.Error("PromptCostEstimation missing arch or cloud")
	}

	if d := PromptDiagramDescription("arch"); !strings.Contains(d, "arch") {
		t.Error("PromptDiagramDescription missing arch")
	}

	if g := PromptTerraformGeneration("arch", "aws"); !strings.Contains(g, "arch") || !strings.Contains(g, "aws") {
		t.Error("PromptTerraformGeneration missing arch or cloud")
	}

	if s := PromptCaseStudyRequest("usecase", "scale"); !strings.Contains(s, "usecase") || !strings.Contains(s, "scale") {
		t.Error("PromptCaseStudyRequest missing usecase or scale")
	}
}

func TestValidateArchitecture(t *testing.T) {
	arch := "mongodb.*transaction no.*backup monolith"
	warnings := ValidateArchitecture(arch)

	expected := []string{
		"⚠️ MongoDB for transactions - consider PostgreSQL",
		"⚠️ No backup strategy - implement automated backups",
		"⚠️ Monolithic architecture - consider microservices for team scale",
	}

	for _, exp := range expected {
		found := false
		for _, w := range warnings {
			if w == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected warning %q not found", exp)
		}
	}
}
