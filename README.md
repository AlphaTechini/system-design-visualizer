# System Design Visualizer

An AI-powered architect that transforms high-level requirements into production-ready system designs, complete with visual diagrams, cost estimates, and Terraform code.

---

## 🚀 Overview

The **System Design Visualizer** bridges the gap between conceptual requirements and technical implementation. By leveraging state-of-the-art LLMs via NEAR AI Cloud and visualizing architectures through Mermaid.js, it provides engineers and stakeholders with an immediate, data-driven starting point for complex distributed systems.

### Core Capabilities
- **AI-Guided Requirements Engineering**: Interactive Q&A to flush out scale, latency, and consistency needs.
- **Visual Architecture Generation**: Real-time rendering of system diagrams using Mermaid.ink.
- **Multi-Cloud Cost Estimation**: Deep analysis of monthly spend across AWS, GCP, and Azure.
- **Infrastructure as Code**: One-click generation of production-grade Terraform modules.
- **Case-Study Validation**: Benchmarking recommendations against real-world architectures (e.g., Netflix, Uber, Instagram).

---

## 🏗️ Architectural Decisions & Tradeoffs

As a senior-level project, every technical choice was made with specific tradeoffs in mind regarding scalability, maintainability, and developer velocity.

### 1. Go for Backend Services
**Decision**: Standard library-heavy Go 1.21+ using `gorilla/mux`.
- **Pros**:
    - **Concurrency**: Exceptional handling of concurrent AI requests and polling via Goroutines.
    - **Performance**: Low memory footprint and fast startup times, ideal for containerized deployments.
    - **Type Safety**: Strong typing reduces runtime errors in complex cost-estimation logic.
- **Tradeoffs**: More verbose than Node.js or Python for simple CRUD, but the long-term maintainability for a distributed system tool outweighs the initial boilerplate.

### 2. SvelteKit for Frontend
**Decision**: SvelteKit 2.x with TypeScript and Tailwind CSS.
- **Pros**:
    - **Reactivity**: Svelte's "no-vdom" approach ensures the UI remains snappy even with large, complex diagrams.
    - **SSR/SSG**: Flexible deployment options to Cloudflare Pages or Vercel.
- **Tradeoffs**: Smaller ecosystem than React, but the developer experience and performance characteristics are superior for data-heavy dashboards.

### 3. NEAR AI Cloud Integration
**Decision**: Utilizing `deepseek-ai/DeepSeek-V3.1` via NEAR AI Cloud.
- **Pros**:
    - **Privacy & Security**: TEE-secured inference ensures that sensitive architectural requirements are handled with higher security guarantees than standard public APIs.
    - **OpenAI Compatibility**: Seamless integration with existing Go SDKs.
- **Tradeoffs**: Newer ecosystem than OpenAI direct, but the privacy-first approach is vital for enterprise system design.

### 4. Mermaid.ink for Visuals
**Decision**: External rendering via `mermaid.ink`.
- **Pros**:
    - **Simplicity**: Decouples the backend from heavy browser-based rendering (Puppeteer/Playwright).
    - **Speed**: Instant conversion of Mermaid DSL into PNG/SVG/PDF via a simple GET request.
- **Tradeoffs**: Dependency on an external service. A local Puppeteer-based fallback is implemented in `internal/diagram` for high-availability scenarios.

---

## 🛠️ Getting Started

### Prerequisites
- **Go**: 1.21 or higher
- **Node.js**: 20.x or higher (for frontend)
- **PostgreSQL**: 15+ (or a Supabase instance)
- **Redis**: 7.x (for caching and rate limiting)

### Environment Configuration
Copy the `.env.example` to `.env` and fill in your credentials:
```bash
cp .env.example .env
```

Key variables:
- `SUPABASE_HOST` / `SUPABASE_PASSWORD`: Database connection details.
- `NEAR_AI_API_KEY`: Obtain from [cloud.near.ai](https://cloud.near.ai).
- `GCP_API_KEY`: Required for fetching real-time cloud pricing.

### Backend Setup
```bash
cd cmd/server
go run main.go
```
The server will start on `http://localhost:8080`.

### Frontend Setup
```bash
cd web
npm install
npm run dev
```
The UI will be available at `http://localhost:5173`.

---

## 🗺️ Roadmap

- [x] **MVP Architecture Recommendation**: Core AI engine integrated with NEAR AI.
- [x] **Visual Diagrams**: Basic Mermaid.js rendering.
- [ ] **Advanced Cost Engine**: Integration with multiple cloud pricing APIs for "What-If" analysis.
- [ ] **Terraform Full-Module Export**: Currently generates fragments; moving to full multi-file modules.
- [ ] **Multi-Cloud Comparison**: Side-by-side architecture comparison for AWS vs. GCP vs. Azure.
- [ ] **User Auth & Persistence**: Move beyond IP-based rate limiting to full user accounts.

---

## 🤝 Contributing

We welcome contributions from the community! Whether it's adding new cloud providers to the cost engine or improving the AI prompts, please follow these steps:

1. Fork the repository.
2. Create a feature branch: `git checkout -b feature/amazing-feature`.
3. Commit your changes: `git commit -m 'Add amazing feature'`.
4. Push to the branch: `git push origin feature/amazing-feature`.
5. Open a Pull Request.

Please ensure all Go code is formatted with `gofmt` and all TypeScript passes `npm run check`.

---

## 📖 Deep Dive

For a detailed look at the system design, database schema, and API specifications, please refer to the [ARCHITECTURE.md](./ARCHITECTURE.md) document.
