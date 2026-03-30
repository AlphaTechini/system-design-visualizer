# System Design Visualizer - Architecture Specification

## Scale & Performance Requirements

**Target Load:**
- 100 concurrent users designing systems
- 500 RPS peak (100 users × 5 API calls per diagram)
- 20 components max per diagram (MVP)
- <2s AI response time for chat
- <30s total diagram generation time

**Consistency Model:**
- Strong consistency: Design saves, user credits
- Eventual consistency: AI response cache, cost estimates

**Failure Tolerance:**
- AI service down → Cancel request, show error (no queue for MVP)
- Database unavailable → Hard fail with 503
- Image generation fails → Retry once, then show Mermaid code fallback

**Delivery Semantics:**
- At-least-once: Design persistence
- At-most-once: AI requests (idempotent via caching)

---

## Technology Stack

### Backend (Go)
```
Runtime: Go 1.21+
Framework: Fastify-like (Gin or Echo for HTTP)
Database: PostgreSQL 15+ (designs, versions, cache metadata)
Cache: Redis 7+ (AI response cache, rate limiting)
Queue: None for MVP (synchronous only)
Storage: Local filesystem for PNG/PDF exports (S3 later)
```

### Frontend (SvelteKit)
```
Framework: SvelteKit 2.x (TypeScript)
Visualization: Mermaid.js (diagrams), jsPDF (PDF export)
UI: Tailwind CSS + Headless UI
State: Svelte stores
Deployment: Cloudflare Pages (static) + API calls to Go backend
```

### External Services
```
AI Provider: NEAR AI Cloud (OpenAI-compatible API, TEE-secured)
  - Endpoint: https://api.near.ai/v1
  - Models: deepseek-ai/DeepSeek-V3.1 (and others)
  - Auth: Bearer token from cloud.near.ai
  
Image Generation: Mermaid.ink (free, no API key needed)
  - URL: https://mermaid.ink/img/{base64_mermaid_code}
  - Formats: PNG, PDF, SVG
  - Alternative: Puppeteer self-hosted fallback
  
Pricing Data: AWS/Azure/GCP pricing APIs (cached hourly)
Rate Limiting: IP-based (1 free/day, +1 for signup, paid tiers)
```

---

## System Architecture

```
┌─────────────────┐
│   User Browser  │
│   (SvelteKit)   │
└────────┬────────┘
         │ HTTPS
         ▼
┌─────────────────┐
│  Cloudflare CDN │ (static assets, caching)
└────────┬────────┘
         │
         ▼
┌─────────────────┐      ┌──────────────┐
│   Go Backend    │◄────►│    Redis     │
│   (Gin/Echo)    │      │ (cache + RL) │
└────────┬────────┘      └──────────────┘
         │
         ├─────────────────┐
         │                 │
         ▼                 ▼
┌─────────────────┐ ┌──────────────┐
│   PostgreSQL    │ │ NEAR AI Cloud│
│  (designs, etc) │ │   (AI API)   │
└─────────────────┘ └──────────────┘
         │
         ▼
┌─────────────────┐
│ Nano Banana Pro │
│  (PNG/PDF gen)  │
└─────────────────┘
```

---

## Database Schema (PostgreSQL)

```sql
-- Rate limiting by IP (no auth for MVP)
CREATE TABLE rate_limits (
    ip_address INET PRIMARY KEY,
    free_generations_used INT DEFAULT 0,
    bonus_generations_used INT DEFAULT 0,
    last_reset_date DATE DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Design sessions
CREATE TABLE designs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ip_address INET NOT NULL,
    name VARCHAR(255) NOT NULL,
    requirements_json JSONB NOT NULL,
    ai_recommendations_json JSONB,
    mermaid_code TEXT,
    terraform_code TEXT,
    cost_estimate_json JSONB,
    status VARCHAR(50) DEFAULT 'draft', -- draft, completed, failed
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Cached AI responses (for identical requirement patterns)
CREATE TABLE ai_cache (
    requirements_hash VARCHAR(64) PRIMARY KEY, -- SHA256 of sorted requirements
    ai_response_json JSONB NOT NULL,
    hit_count INT DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_designs_ip ON designs(ip_address);
CREATE INDEX idx_designs_status ON designs(status);
CREATE INDEX idx_ai_cache_last_used ON ai_cache(last_used_at);

-- Daily reset job (run at midnight UTC)
-- UPDATE rate_limits SET free_generations_used = 0, last_reset_date = CURRENT_DATE;
```

---

## API Endpoints (Go Backend)

### 1. Requirements Intake
```
POST /api/v1/designs
Request:
{
  "expected_users_month_1": 1000,
  "expected_users_year_1": 50000,
  "expected_users_year_3": 500000,
  "peak_rps": 5000,
  "latency_budget_ms": 100,
  "data_retention_days": 365,
  "compliance": ["GDPR"],
  "read_write_ratio": "80:20",
  "consistency": "eventual",
  "budget_monthly_usd": 5000
}

Response:
{
  "design_id": "uuid",
  "status": "requirements_received",
  "clarifying_questions": [
    "Are 500K users MAU or concurrent?",
    "For 'real-time' - is <100ms acceptable?"
  ]
}
```

### 2. AI Q&A Session
```
POST /api/v1/designs/{id}/chat
Request:
{
  "message": "Why not just use MongoDB?",
  "context": {"previous_recommendation": "PostgreSQL + TimescaleDB"}
}

Response:
{
  "answer": "MongoDB simpler but you'll hit sharding limits at 10M documents...",
  "trade_offs": [
    {"option": "MongoDB", "pros": ["Easy scaling"], "cons": ["Eventual consistency only"]},
    {"option": "PostgreSQL", "pros": ["ACID compliance"], "cons": ["Sharding complexity"]}
  ],
  "case_study": "Instagram scaled PostgreSQL to billions of rows with partitioning"
}
```

### 3. Generate Diagram
```
POST /api/v1/designs/{id}/diagram
Request: {} (uses existing requirements + AI recommendations)

Response (async polling):
{
  "status": "processing", -- or "completed", "failed"
  "mermaid_code": "graph TD...",
  "png_url": "/api/v1/designs/{id}/diagram.png",
  "pdf_url": "/api/v1/designs/{id}/diagram.pdf"
}

GET /api/v1/designs/{id}/diagram.png
→ Returns PNG image (generated by Nano Banana Pro API)
```

### 4. Cost Estimation
```
POST /api/v1/designs/{id}/cost
Request: {} (uses AI-recommended architecture)

Response:
{
  "monthly_breakdown": {
    "compute": 2500,
    "database": 1200,
    "storage": 300,
    "network": 500,
    "hidden_costs": {
      "data_transfer_cross_az": 200,
      "api_calls_cloudwatch": 50
    }
  },
  "total_monthly": 4750,
  "scaling_projection": {
    "10x_users": 12000,
    "100x_users": 45000
  },
  "cloud_provider": "AWS",
  "comparison": {
    "GCP": 4900,
    "Azure": 5100
  }
}
```

### 5. Terraform Generation
```
POST /api/v1/designs/{id}/terraform
Request: {
  "confirmations": {
    "database_instance": "db.r5.large",
    "cache_nodes": 3,
    "enable_read_replicas": true
  }
}

Response:
{
  "terraform_code": "resource \"aws_db_instance\" \"main\" {...}",
  "files": [
    {"name": "main.tf", "content": "..."},
    {"name": "variables.tf", "content": "..."},
    {"name": "outputs.tf", "content": "..."}
  ]
}
```

### 6. Rate Limit Check
```
GET /api/v1/rate-limit
Response (from IP):
{
  "free_remaining": 1,
  "bonus_remaining": 0,
  "reset_at": "2026-02-22T00:00:00Z",
  "upgrade_url": "/pricing"
}
```

---

## Rate Limiting Logic

```go
// Middleware: CheckRateLimit
func CheckRateLimit(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := getClientIP(r)
        
        // Get or create rate limit record
        rl := db.GetRateLimit(ip)
        
        // Reset if new day
        if rl.LastResetDate < today {
            rl.FreeGenerationsUsed = 0
            rl.BonusGenerationsUsed = 0
            rl.LastResetDate = today
        }
        
        // Check limits
        if rl.FreeGenerationsUsed >= 1 && rl.BonusGenerationsUsed >= 1 {
            // Offer upgrade
            w.WriteHeader(http.StatusTooManyRequests)
            json.NewEncoder(w).Encode(map[string]interface{}{
                "error": "Daily limit reached",
                "upgrade_url": "/pricing",
                "retry_after": "tomorrow 00:00 UTC"
            })
            return
        }
        
        // Increment counter
        if rl.FreeGenerationsUsed < 1 {
            rl.FreeGenerationsUsed++
        } else {
            rl.BonusGenerationsUsed++
        }
        db.SaveRateLimit(rl)
        
        next.ServeHTTP(w, r)
    })
}
```

---

## AI Integration (NEAR AI Cloud - OpenAI Compatible)

**API Details:**
- Base URL: `https://api.near.ai/v1`
- Auth: `Authorization: Bearer <token>` (from cloud.near.ai)
- Models: `deepseek-ai/DeepSeek-V3.1` (default), others available
- Compatibility: OpenAI SDK works out-of-the-box

```go
type NEARAIProvider struct {
    APIKey     string
    BaseURL    string // https://api.near.ai/v1
    Model      string // deepseek-ai/DeepSeek-V3.1
    Timeout    time.Duration
    httpClient *http.Client
}

func NewNEARAIProvider(apiKey string) *NEARAIProvider {
    return &NEARAIProvider{
        APIKey:  apiKey,
        BaseURL: "https://api.near.ai/v1",
        Model:   "deepseek-ai/DeepSeek-V3.1",
        Timeout: 30 * time.Second,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// OpenAI-compatible request/response structures
type ChatCompletionRequest struct {
    Model       string    `json:"model"`
    Messages    []Message `json:"messages"`
    Temperature float64   `json:"temperature,omitempty"`
    MaxTokens   int       `json:"max_tokens,omitempty"`
}

type Message struct {
    Role    string `json:"role"` // "system", "user", "assistant"
    Content string `json:"content"`
}

type ChatCompletionResponse struct {
    ID      string `json:"id"`
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

func (n *NEARAIProvider) Chat(ctx context.Context, systemPrompt, userMessage string) (string, error) {
    // Check cache first
    cacheKey := "ai:" + sha256sum(systemPrompt+userMessage)
    if cached, ok := redis.Get(ctx, cacheKey); ok {
        return cached, nil
    }
    
    req := ChatCompletionRequest{
        Model: n.Model,
        Messages: []Message{
            {Role: "system", Content: systemPrompt},
            {Role: "user", Content: userMessage},
        },
        Temperature: 0.7,
        MaxTokens:   2000,
    }
    
    reqBody, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", 
        n.BaseURL+"/chat/completions", bytes.NewReader(reqBody))
    
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+n.APIKey)
    
    resp, err := n.httpClient.Do(httpReq)
    if err != nil {
        return "", fmt.Errorf("NEAR AI Cloud error: %w", err)
    }
    defer resp.Body.Close()
    
    var completion ChatCompletionResponse
    if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
        return "", fmt.Errorf("decode error: %w", err)
    }
    
    if len(completion.Choices) == 0 {
        return "", fmt.Errorf("no choices in response")
    }
    
    answer := completion.Choices[0].Message.Content
    
    // Cache for 24 hours
    redis.Set(ctx, cacheKey, answer, 24*time.Hour)
    
    return answer, nil
}

var SystemPrompt = `You are an expert system architect specializing in large-scale distributed systems.

Your role:
1. Ask clarifying questions about scale, latency, consistency requirements
2. Recommend architectures based on CAP theorem trade-offs
3. Reference real case studies (Netflix, Uber, Airbnb, etc.)
4. Warn about anti-patterns and single points of failure
5. Include hidden costs in estimates (data transfer, cross-AZ traffic, API calls)
6. Block obviously bad architectures (e.g., MongoDB for billion-row transactional workloads)

Always state:
- Your recommended cloud provider and why
- Specific instance types and sizes
- Expected monthly costs with breakdown
- Scaling bottlenecks and when to reconsider architecture
- Relevant case studies from similar companies`
```

---

## Image Generation (Mermaid.ink - Free, No API Key)

**Service:** https://mermaid.ink  
**Usage:** `GET https://mermaid.ink/img/{base64_encoded_mermaid}`  
**Formats:** PNG (default), PDF (`/pdf/`), SVG (`/svg/`)

```go
type MermaidRenderer struct {
    BaseURL string // https://mermaid.ink
    client  *http.Client
}

func NewMermaidRenderer() *MermaidRenderer {
    return &MermaidRenderer{
        BaseURL: "https://mermaid.ink",
        client:  &http.Client{Timeout: 30 * time.Second},
    }
}

// RenderPNG converts Mermaid code to PNG bytes
func (m *MermaidRenderer) RenderPNG(mermaidCode string) ([]byte, error) {
    // Encode mermaid code as base64
    encoded := base64.StdEncoding.EncodeToString([]byte(mermaidCode))
    
    // URL format: https://mermaid.ink/img/{base64_code}
    url := fmt.Sprintf("%s/img/%s", m.BaseURL, encoded)
    
    resp, err := m.client.Get(url)
    if err != nil {
        return nil, fmt.Errorf("mermaid.ink error: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("render failed with status %d", resp.StatusCode)
    }
    
    return io.ReadAll(resp.Body)
}

// RenderPDF converts Mermaid code to PDF bytes
func (m *MermaidRenderer) RenderPDF(mermaidCode string) ([]byte, error) {
    encoded := base64.StdEncoding.EncodeToString([]byte(mermaidCode))
    url := fmt.Sprintf("%s/pdf/%s", m.BaseURL, encoded)
    
    resp, err := m.client.Get(url)
    if err != nil {
        return nil, fmt.Errorf("mermaid.ink PDF error: %w", err)
    }
    defer resp.Body.Close()
    
    return io.ReadAll(resp.Body)
}
```

**Fallback:** If mermaid.ink is down, use Puppeteer locally:
```go
// Fallback implementation using chromedp
func RenderWithPuppeteer(mermaidCode string) ([]byte, error) {
    // Launch headless Chrome, render Mermaid, screenshot
    // Implementation in diagram/renderer_fallback.go
}
```

---

## Directory Structure

```
system-design-visualizer/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── api/
│   │   ├── handlers.go          # HTTP handlers
│   │   ├── middleware.go        # Rate limiting, CORS, logging
│   │   └── routes.go            # Route definitions
│   ├── database/
│   │   ├── postgres.go          # DB connection, migrations
│   │   └── queries.sql          # SQL queries
│   ├── redis/
│   │   └── client.go            # Redis client, cache helpers
│   ├── ai/
│   │   ├── near.go              # NEAR AI Cloud provider
│   │   ├── prompt.go            # System prompts, templates
│   │   └── cache.go             # AI response caching
│   ├── diagram/
│   │   ├── mermaid.go           # Mermaid code generation
│   │   └── renderer.go          # Nano Banana Pro integration
│   ├── cost/
│   │   ├── estimator.go         # Cost calculation engine
│   │   └── providers.go         # AWS/GCP/Azure pricing APIs
│   ├── terraform/
│   │   ├── generator.go         # Hybrid TF generation
│   │   └── templates/           # TF template fragments
│   └── ratelimit/
│       └── middleware.go        # IP-based rate limiting
├── pkg/
│   ├── models/
│   │   ├── design.go            # Design structs
│   │   └── ratelimit.go         # Rate limit structs
│   └── config/
│       └── config.go            # Environment config
├── web/                         # SvelteKit frontend
│   ├── src/
│   │   ├── routes/
│   │   ├── lib/
│   │   └── app.css
│   └── static/
├── migrations/
│   └── 001_initial_schema.sql
├── .env.example
├── go.mod
├── go.sum
└── README.md
```

---

## Deployment (Single VPS)

```bash
# Ubuntu 22.04 LTS VM (4 vCPU, 8GB RAM, 100GB SSD)

# Install Go
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz

# Install PostgreSQL
sudo apt install postgresql postgresql-contrib

# Install Redis
sudo apt install redis-server

# Build Go binary
cd system-design-visualizer
go build -o bin/server ./cmd/server

# Systemd service
sudo nano /etc/systemd/system/sdv-server.service
# [Unit]
# Description=System Design Visualizer Backend
# After=network.target postgresql.service redis.service
# 
# [Service]
# Type=simple
# User=sdv
# WorkingDirectory=/opt/sdv
# ExecStart=/opt/sdv/bin/server
# Restart=always
# Environment=DATABASE_URL=postgres://...
# Environment=REDIS_URL=redis://localhost:6379
# 
# [Install]
# WantedBy=multi-user.target

sudo systemctl enable sdv-server
sudo systemctl start sdv-server

# Nginx reverse proxy
sudo nano /etc/nginx/sites-available/sdv
# server {
#     listen 80;
#     server_name your-domain.com;
#     
#     location / {
#         proxy_pass http://localhost:8080;
#         proxy_set_header Host $host;
#         proxy_set_header X-Real-IP $remote_addr;
#     }
# }

sudo ln -s /etc/nginx/sites-available/sdv /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```


