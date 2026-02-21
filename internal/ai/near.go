package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// NEARAIProvider implements OpenAI-compatible API for NEAR AI Cloud
type NEARAIProvider struct {
	apiKey    string
	baseURL   string
	model     string
	timeout   time.Duration
	httpClient *http.Client
}

// ChatCompletionRequest matches OpenAI API structure
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// ChatCompletionResponse matches OpenAI API response
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int     `json:"index"`
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewNEARAIProvider creates NEAR AI Cloud client
func NewNEARAIProvider(apiKey, model string) *NEARAIProvider {
	if model == "" {
		model = "deepseek-ai/DeepSeek-V3.1"
	}

	return &NEARAIProvider{
		apiKey:  apiKey,
		baseURL: "https://api.near.ai/v1",
		model:   model,
		timeout: 30 * time.Second,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Chat sends a chat completion request with caching
func (n *NEARAIProvider) Chat(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	// Generate cache key from both prompts
	cacheKey := n.cacheKey(systemPrompt, userMessage)

	// Try cache first (Redis would be here, using memory for now)
	if cached, ok := memoryCache.Get(cacheKey); ok {
		return cached, nil
	}

	// Build request
	req := ChatCompletionRequest{
		Model: n.model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		n.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+n.apiKey)

	// Execute request
	resp, err := n.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("NEAR AI Cloud error: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var completion ChatCompletionResponse
	if err := json.Unmarshal(body, &completion); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	answer := completion.Choices[0].Message.Content

	// Cache for 24 hours
	memoryCache.Set(cacheKey, answer, 24*time.Hour)

	return answer, nil
}

// cacheKey generates SHA256 hash of combined prompts
func (n *NEARAIProvider) cacheKey(systemPrompt, userMessage string) string {
	combined := systemPrompt + "|_|" + userMessage
	hash := sha256.Sum256([]byte(combined))
	return "ai:" + hex.EncodeToString(hash[:])
}

// Simple in-memory cache (replace with Redis in production)
type MemoryCache struct {
	data map[string]cacheEntry
}

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

var memoryCache = &MemoryCache{
	data: make(map[string]cacheEntry),
}

func (c *MemoryCache) Get(key string) (string, bool) {
	entry, ok := c.data[key]
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			delete(c.data, key)
		}
		return "", false
	}
	return entry.value, true
}

func (c *MemoryCache) Set(key, value string, ttl time.Duration) {
	c.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}
