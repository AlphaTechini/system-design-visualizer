package ai
import "fmt"

// SystemPromptArchitect is the main system prompt for architecture advice
const SystemPromptArchitect = `You are an expert system architect specializing in large-scale distributed systems.

YOUR ROLE:
1. Ask clarifying questions about scale, latency, consistency, and compliance requirements
2. Recommend architectures based on CAP theorem trade-offs
3. Reference real case studies (Netflix, Uber, Airbnb, Instagram, etc.)
4. Warn about anti-patterns and single points of failure
5. Include hidden costs in estimates (data transfer, cross-AZ traffic, API calls)
6. BLOCK obviously bad architectures with clear explanations

ALWAYS STATE:
- Your recommended cloud provider (AWS/GCP/Azure) and WHY
- Specific instance types and sizes (e.g., db.r5.large, not just "RDS")
- Expected monthly costs with detailed breakdown
- Scaling bottlenecks and when to reconsider architecture
- Relevant case studies from similar companies/scale

ANTI-PATTERNS TO BLOCK:
❌ MongoDB for billion-row transactional workloads (suggest PostgreSQL + TimescaleDB)
❌ Single database for mixed OLTP + analytics (suggest CQRS)
❌ Synchronous microservices with tight coupling (suggest event-driven)
❌ No caching layer for read-heavy workloads (>80% reads)
❌ Monolithic deployment for teams >10 engineers
❌ No circuit breakers for external API calls
❌ Database on same server as application
❌ No backup/disaster recovery strategy

CAP THEOREM GUIDANCE:
- Strong consistency needed? → PostgreSQL, CockroachDB, Spanner
- Eventual consistency OK? → Cassandra, DynamoDB, MongoDB
- Low latency critical? → Redis, Memcached, CDN
- Global distribution? → Multi-region with conflict resolution

COST OPTIMIZATION TIPS:
- Use reserved instances for steady-state workloads (save 40-60%)
- Spot instances for stateless workers (save 70-90%)
- S3 Intelligent Tiering for unpredictable access patterns
- CloudFront/CDN for static assets (reduce origin load 80%+)
- Read replicas before sharding (cheaper complexity)

CASE STUDY EXAMPLES:
- Instagram: PostgreSQL partitioning for billions of rows
- Netflix: Chaos Engineering + multi-region resilience
- Uber: Schemaless (MySQL abstraction) for horizontal scaling
- Airbnb: Druid for real-time analytics at scale
- Twitter: Manhattan (distributed NoSQL) for timeline storage

RESPONSE FORMAT:
1. Clarifying Questions (if requirements unclear)
2. Recommended Architecture (with diagram description)
3. Technology Choices (with alternatives considered)
4. Cost Estimate (monthly breakdown + hidden costs)
5. Scaling Bottlenecks (when to revisit)
6. Relevant Case Study
7. Risk Assessment (SPOFs, mitigation strategies)`

// PromptClarifyingQuestions generates follow-up questions
func PromptClarifyingQuestions(requirements map[string]interface{}) string {
	return `Based on these requirements, I need clarification:

` + formatRequirements(requirements) + `

Please clarify:
1. Are the user counts MAU (monthly active) or concurrent users?
2. What's your latency budget? (<100ms, <500ms, <1s, or batch OK?)
3. Do you need strong consistency or is eventual consistency acceptable?
4. Any specific compliance requirements (GDPR, HIPAA, SOC2)?
5. What's your team's existing tech stack expertise?
6. Budget constraints (monthly infrastructure spend)?
7. Expected growth rate (month-over-month)?
8. Geographic distribution (single region, multi-region, global)?`
}

// PromptArchitectureRecommendation generates full architecture advice
func PromptArchitectureRecommendation(requirements map[string]interface{}, clarifications string) string {
	return `Given these requirements and clarifications:

REQUIREMENTS:
` + formatRequirements(requirements) + `

CLARIFICATIONS:
` + clarifications + `

Please provide:
1. Complete architecture recommendation
2. Specific technology choices with versions
3. Instance types and quantities
4. Monthly cost breakdown
5. Scaling strategy
6. Relevant case study
7. Risk assessment`
}

// PromptCostEstimation requests detailed cost analysis
func PromptCostEstimation(architecture string, cloudProvider string) string {
	return `For this architecture on ` + cloudProvider + `:

` + architecture + `

Provide detailed monthly cost estimate including:
1. Compute costs (instance types × quantity × hours)
2. Database costs (storage + IOPS + backups)
3. Network costs (data transfer, cross-AZ, internet egress)
4. Storage costs (S3/blob storage tiers)
5. Hidden costs (CloudWatch, API calls, NAT Gateway, Load Balancer hours)
6. Reserved instance vs on-demand comparison
7. Scaling cost projection (10x users, 100x users)

Format as JSON with line items.`
}

// PromptDiagramDescription generates Mermaid diagram code
func PromptDiagramDescription(architecture string) string {
	return `Convert this architecture into a Mermaid flowchart (graph TD):

` + architecture + `

Include:
- User/Client at top
- Load balancers
- Application services
- Databases (primary + replicas)
- Cache layer (Redis/Memcached)
- Message queues (Kafka/RabbitMQ/SQS)
- External APIs
- Monitoring/Logging stack

Use proper Mermaid syntax with subgraphs for logical grouping.
Make it readable for 20 components max.`
}

// PromptTerraformGeneration creates infrastructure code request
func PromptTerraformGeneration(architecture string, cloudProvider string) string {
	return `Generate Terraform code for this ` + cloudProvider + ` architecture:

` + architecture + `

Include:
1. VPC with public/private subnets
2. Security groups with least-privilege rules
3. RDS/database cluster with backups
4. ElastiCache/Memorystore cluster
5. Auto Scaling Group with launch template
6. Application Load Balancer
7. S3 buckets with lifecycle policies
8. CloudWatch alarms and dashboards
9. IAM roles with minimal permissions

Output as separate files:
- main.tf (resources)
- variables.tf (input variables)
- outputs.tf (exported values)
- providers.tf (provider configuration)`
}

// PromptCaseStudyRequest asks for relevant examples
func PromptCaseStudyRequest(useCase string, scale string) string {
	return `Find a relevant case study for:

Use Case: ` + useCase + `
Scale: ` + scale + `

Include:
1. Company name and industry
2. Their architecture choices
3. Problems they solved
4. Lessons learned
5. Links to engineering blog posts if available`
}

// formatRequirements converts map to readable text
func formatRequirements(reqs map[string]interface{}) string {
	result := ""
	for key, value := range reqs {
		result += "- " + key + ": " + fmt.Sprintf("%v", value) + "\n"
	}
	return result
}

// ValidateArchitecture checks for anti-patterns
func ValidateArchitecture(architecture string) []string {
	warnings := []string{}

	// Check for common anti-patterns
	antiPatterns := map[string]string{
		"mongodb.*transaction":     "⚠️ MongoDB for transactions - consider PostgreSQL",
		"single.*database":         "⚠️ Single database - consider read replicas or sharding",
		"no.*cache":                "⚠️ No caching layer - add Redis/Memcached for read-heavy workloads",
		"monolith":                 "⚠️ Monolithic architecture - consider microservices for team scale",
		"no.*backup":               "⚠️ No backup strategy - implement automated backups",
		"synchronous.*microservice": "⚠️ Synchronous microservices - consider event-driven architecture",
	}

	for pattern, warning := range antiPatterns {
		// Simple string matching (use regex in production)
		if containsIgnoreCase(architecture, pattern) {
			warnings = append(warnings, warning)
		}
	}

	return warnings
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || 
		 len(s) > len(substr) && 
		 (s[:len(substr)] == substr || 
		  s[len(s)-len(substr):] == substr || 
		  containsIgnoreCaseHelper(s, substr)))
}

func containsIgnoreCaseHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
