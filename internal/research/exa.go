package research

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ExaClient implements Exa AI search API
type ExaClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// SearchRequest matches Exa API structure
type SearchRequest struct {
	Query          string            `json:"query"`
	Type           string            `json:"type,omitempty"` // "auto", "fast"
	Category       string            `json:"category,omitempty"` // "people", "company", "news", "research paper", "tweet"
	NumResults     int               `json:"num_results,omitempty"`
	IncludeDomains []string          `json:"includeDomains,omitempty"`
	ExcludeDomains []string          `json:"excludeDomains,omitempty"`
	Contents       ContentConfig     `json:"contents,omitempty"`
	MaxAgeHours    int               `json:"maxAgeHours,omitempty"`
}

// ContentConfig specifies what content to return
type ContentConfig struct {
	Text      *TextConfig      `json:"text,omitempty"`
	Highlights *HighlightsConfig `json:"highlights,omitempty"`
}

// TextConfig for full page extraction
type TextConfig struct {
	MaxCharacters int `json:"max_characters,omitempty"`
}

// HighlightsConfig for snippets
type HighlightsConfig struct {
	MaxCharacters int `json:"max_characters,omitempty"`
}

// SearchResponse matches Exa API response
type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

// SearchResult represents a single result
type SearchResult struct {
	Title string  `json:"title"`
	URL   string  `json:"url"`
	Score float64 `json:"score"`
	Text  string  `json:"text,omitempty"`
}

// NewExaClient creates Exa API client
func NewExaClient(apiKey string) *ExaClient {
	return &ExaClient{
		apiKey:  apiKey,
		baseURL: "https://api.exa.ai",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Search performs a general search
func (e *ExaClient) Search(query string, numResults int) ([]SearchResult, error) {
	req := SearchRequest{
		Query:      query,
		Type:       "auto",
		NumResults: numResults,
		Contents: ContentConfig{
			Text: &TextConfig{
				MaxCharacters: 20000,
			},
		},
	}

	return e.search(req)
}

// SearchPeople finds people by role/expertise
func (e *ExaClient) SearchPeople(query string, numResults int) ([]SearchResult, error) {
	req := SearchRequest{
		Query:      query,
		Category:   "people",
		Type:       "auto",
		NumResults: numResults,
	}

	return e.search(req)
}

// SearchCompanies finds companies by industry/criteria
func (e *ExaClient) SearchCompanies(query string, numResults int) ([]SearchResult, error) {
	req := SearchRequest{
		Query:      query,
		Category:   "company",
		Type:       "auto",
		NumResults: numResults,
	}

	return e.search(req)
}

// SearchNews finds recent news articles
func (e *ExaClient) SearchNews(query string, numResults int, maxAgeHours int) ([]SearchResult, error) {
	req := SearchRequest{
		Query:       query,
		Category:    "news",
		Type:        "auto",
		NumResults:  numResults,
		MaxAgeHours: maxAgeHours,
		Contents: ContentConfig{
			Text: &TextConfig{
				MaxCharacters: 20000,
			},
		},
	}

	return e.search(req)
}

// SearchResearchPapers finds academic papers
func (e *ExaClient) SearchResearchPapers(query string, numResults int) ([]SearchResult, error) {
	req := SearchRequest{
		Query:      query,
		Category:   "research paper",
		Type:       "auto",
		NumResults: numResults,
		Contents: ContentConfig{
			Text: &TextConfig{
				MaxCharacters: 20000,
			},
		},
	}

	return e.search(req)
}

// search is the internal implementation
func (e *ExaClient) search(req SearchRequest) ([]SearchResult, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", e.baseURL+"/search", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", e.apiKey)

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Exa API error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var searchResp SearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return searchResp.Results, nil
}

// FindCaseStudies searches for relevant architecture case studies
func (e *ExaClient) FindCaseStudies(useCase string, scale string, technologies []string) ([]SearchResult, error) {
	query := fmt.Sprintf("%s architecture at %s scale", useCase, scale)
	if len(technologies) > 0 {
		query += fmt.Sprintf(" using %v", technologies)
	}
	query += " engineering blog"

	return e.Search(query, 5)
}

// FindSimilarCompanies finds companies with similar tech stack
func (e *ExaClient) FindSimilarCompanies(techStack []string, industry string) ([]SearchResult, error) {
	query := fmt.Sprintf("company using %v", techStack)
	if industry != "" {
		query += fmt.Sprintf(" in %s industry", industry)
	}

	return e.SearchCompanies(query, 10)
}

// FindExperts finds people with relevant expertise
func (e *ExaClient) FindExperts(expertise string, technologies []string) ([]SearchResult, error) {
	query := fmt.Sprintf("engineer architect %s", expertise)
	if len(technologies) > 0 {
		query += fmt.Sprintf(" %v", technologies)
	}

	return e.SearchPeople(query, 10)
}

// ValidateTechnologyChoice searches for real-world usage patterns
func (e *ExaClient) ValidateTechnologyChoice(technology string, useCase string, scale string) ([]SearchResult, error) {
	query := fmt.Sprintf("%s for %s at scale %s pros cons", technology, useCase, scale)
	return e.Search(query, 10)
}

// GetContents fetches full text from known URLs
func (e *ExaClient) GetContents(urls []string, maxCharacters int) ([]ContentResult, error) {
	req := ContentsRequest{
		URLs: urls,
		Text: &TextConfig{
			MaxCharacters: maxCharacters,
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", e.baseURL+"/contents", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", e.apiKey)

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Exa contents error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var contentsResp ContentsResponse
	if err := json.Unmarshal(body, &contentsResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return contentsResp.Results, nil
}

// AnswerQuestion performs Q&A with web citations
func (e *ExaClient) AnswerQuestion(question string, numResults int) (*AnswerResult, error) {
	req := AnswerRequest{
		Query:      question,
		NumResults: numResults,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", e.baseURL+"/answer", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", e.apiKey)

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Exa answer error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var answerResp AnswerResponse
	if err := json.Unmarshal(body, &answerResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &answerResp.Answer, nil
}

// ContentsRequest for /contents endpoint
type ContentsRequest struct {
	URLs  []string      `json:"urls"`
	Text  *TextConfig   `json:"text,omitempty"`
}

// ContentsResponse matches /contents response
type ContentsResponse struct {
	Results []ContentResult `json:"results"`
}

// ContentResult represents extracted content from a URL
type ContentResult struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// AnswerRequest for /answer endpoint
type AnswerRequest struct {
	Query      string `json:"query"`
	NumResults int    `json:"num_results,omitempty"`
}

// AnswerResponse matches /answer response
type AnswerResponse struct {
	Answer AnswerResult `json:"answer"`
}

// AnswerResult contains Q&A with citations
type AnswerResult struct {
	Answer    string   `json:"answer"`
	Sources   []Source `json:"sources"`
}

// Source represents a citation source
type Source struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}
