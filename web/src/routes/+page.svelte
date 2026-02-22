<script lang="ts">
  import { onMount } from 'svelte';
  import Header from '../lib/components/layout/Header.svelte';
  import Footer from '../lib/components/layout/Footer.svelte';
  import URLDropzone from '../lib/components/forms/URLDropzone.svelte';
  import Button from '../lib/components/ui/Button.svelte';
  import Card from '../lib/components/ui/Card.svelte';
  import LoadingSpinner from '../lib/components/ui/LoadingSpinner.svelte';

  type ComplianceOption = 'GDPR' | 'HIPAA' | 'SOC2' | 'PCI-DSS';
  type ConsistencyModel = 'strong' | 'eventual' | 'causal';
  type TrafficPattern = 'constant' | 'ramp-up' | 'burst' | 'seasonal';

  interface DesignRequirements {
    expected_users_month_1: number;
    expected_users_year_1: number;
    expected_users_year_3: number;
    peak_rps: number;
    latency_budget_ms: number;
    data_retention_days: number;
    compliance: ComplianceOption[];
    read_write_ratio: string;
    consistency: ConsistencyModel;
    budget_monthly_usd: number;
  }

  let requirements: Partial<DesignRequirements> = {
    expected_users_month_1: 1000,
    expected_users_year_1: 50000,
    expected_users_year_3: 500000,
    peak_rps: 5000,
    latency_budget_ms: 100,
    data_retention_days: 365,
    compliance: [],
    read_write_ratio: '80:20',
    consistency: 'eventual',
    budget_monthly_usd: 5000
  };

  let clarifyingQuestions: string[] = [];
  let isGenerating = false;
  let designId: string | null = null;
  let mermaidCode: string | null = null;
  let aiRecommendations: string[] = [];
  let costEstimate: any = null;

  const complianceOptions: ComplianceOption[] = ['GDPR', 'HIPAA', 'SOC2', 'PCI-DSS'];
  const consistencyOptions: ConsistencyModel[] = ['strong', 'eventual', 'causal'];

  function toggleCompliance(option: ComplianceOption) {
    if (!requirements.compliance) requirements.compliance = [];
    const idx = requirements.compliance.indexOf(option);
    if (idx > -1) {
      requirements.compliance = requirements.compliance.filter(c => c !== option);
    } else {
      requirements.compliance = [...requirements.compliance, option];
    }
  }

  async function submitRequirements() {
    isGenerating = true;
    clarifyingQuestions = [];
    
    try {
      const response = await fetch('/api/v1/designs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(requirements)
      });

      const data = await response.json();
      designId = data.design_id;
      clarifyingQuestions = data.clarifying_questions || [];
      
      if (clarifyingQuestions.length === 0) {
        await generateDiagram();
      }
    } catch (error) {
      console.error('Failed to submit requirements:', error);
      alert('Failed to submit requirements. Please try again.');
    } finally {
      isGenerating = false;
    }
  }

  async function generateDiagram() {
    if (!designId) return;
    
    isGenerating = true;
    
    try {
      const response = await fetch(`/api/v1/designs/${designId}/diagram`, {
        method: 'POST'
      });

      const data = await response.json();
      mermaidCode = data.mermaid_code;
      aiRecommendations = data.ai_recommendations || [];
      costEstimate = data.cost_estimate;
      
      // Render diagram using Mermaid.js
      renderMermaidDiagram(mermaidCode);
    } catch (error) {
      console.error('Failed to generate diagram:', error);
      alert('Failed to generate diagram. Please try again.');
    } finally {
      isGenerating = false;
    }
  }

  function renderMermaidCode(code: string) {
    // Mermaid.js will be loaded dynamically
    const container = document.getElementById('mermaid-container');
    if (container) {
      container.innerHTML = `<div class="mermaid">${code}</div>`;
      // @ts-ignore
      if (window.mermaid) {
        // @ts-ignore
        window.mermaid.init(undefined, container.querySelectorAll('.mermaid'));
      }
    }
  }

  onMount(async () => {
    // Load Mermaid.js dynamically
    const script = document.createElement('script');
    script.src = 'https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js';
    script.onload = () => {
      // @ts-ignore
      window.mermaid.initialize({ startOnLoad: false, theme: 'default' });
    };
    document.head.appendChild(script);

    // Load saved requirements
    const saved = localStorage.getItem('sdv-requirements');
    if (saved) {
      requirements = JSON.parse(saved);
    }
  });

  function saveRequirements() {
    localStorage.setItem('sdv-requirements', JSON.stringify(requirements));
  }
</script>

<svelte:head>
  <title>System Design Visualizer - AI-Powered Architecture Design</title>
  <meta name="description" content="Transform your requirements into production-ready system architecture diagrams with AI-powered recommendations" />
</svelte:head>

<div class="min-h-screen bg-gray-50">
  <Header />

  <main class="max-w-7xl mx-auto px-4 py-8">
    <!-- Hero Section -->
    {#if !designId}
      <section class="text-center mb-12">
        <h1 class="text-4xl font-bold text-gray-900 mb-4">
          🏗️ System Design Visualizer
        </h1>
        <p class="text-xl text-gray-600 max-w-3xl mx-auto">
          Transform your requirements into production-ready architecture diagrams. 
          Get AI-powered recommendations, cost estimates, and infrastructure code.
        </p>
      </section>
    {/if}

    <!-- Requirements Form -->
    {#if !designId}
      <Card class="mb-8">
        <h2 class="text-2xl font-semibold mb-6">Define Your Requirements</h2>
        
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1" for="users-month-1">
              Users (Month 1)
            </label>
            <input
              id="users-month-1"
              type="number"
              bind:value={requirements.expected_users_month_1}
              class="input w-full"
              onchange={saveRequirements}
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1" for="users-year-1">
              Users (Year 1)
            </label>
            <input
              id="users-year-1"
              type="number"
              bind:value={requirements.expected_users_year_1}
              class="input w-full"
              onchange={saveRequirements}
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1" for="users-year-3">
              Users (Year 3)
            </label>
            <input
              id="users-year-3"
              type="number"
              bind:value={requirements.expected_users_year_3}
              class="input w-full"
              onchange={saveRequirements}
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1" for="peak-rps">
              Peak RPS
            </label>
            <input
              id="peak-rps"
              type="number"
              bind:value={requirements.peak_rps}
              class="input w-full"
              onchange={saveRequirements}
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1" for="latency">
              Latency Budget (ms)
            </label>
            <input
              id="latency"
              type="number"
              bind:value={requirements.latency_budget_ms}
              class="input w-full"
              onchange={saveRequirements}
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1" for="retention">
              Data Retention (days)
            </label>
            <input
              id="retention"
              type="number"
              bind:value={requirements.data_retention_days}
              class="input w-full"
              onchange={saveRequirements}
            />
          </div>
        </div>

        <div class="mb-6">
          <span id="compliance-label" class="block text-sm font-medium text-gray-700 mb-2">Compliance Requirements</span>
          <div class="flex flex-wrap gap-3" role="group" aria-labelledby="compliance-label">
            {#each complianceOptions as option}
              <button
                type="button"
                class="px-4 py-2 rounded-lg border-2 transition-colors"
                class:border-blue-500={requirements.compliance?.includes(option)}
                class:bg-blue-50={requirements.compliance?.includes(option)}
                class:border-gray-300={!requirements.compliance?.includes(option)}
                onclick={() => toggleCompliance(option)}
              >
                {option}
              </button>
            {/each}
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1" for="read-write">
              Read/Write Ratio
            </label>
            <select
              id="read-write"
              bind:value={requirements.read_write_ratio}
              class="input w-full"
              onchange={saveRequirements}
            >
              <option>80:20</option>
              <option>90:10</option>
              <option>70:30</option>
              <option>50:50</option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1" for="consistency">
              Consistency Model
            </label>
            <select
              id="consistency"
              bind:value={requirements.consistency}
              class="input w-full"
              onchange={saveRequirements}
            >
              {#each consistencyOptions as option}
                <option value={option}>{option.charAt(0).toUpperCase() + option.slice(1)}</option>
              {/each}
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1" for="budget">
              Monthly Budget ($)
            </label>
            <input
              id="budget"
              type="number"
              bind:value={requirements.budget_monthly_usd}
              class="input w-full"
              onchange={saveRequirements}
            />
          </div>
        </div>

        <div class="flex gap-4">
          <Button variant="primary" size="lg" onclick={submitRequirements} disabled={isGenerating}>
            {#if isGenerating}
              <LoadingSpinner size="sm" />
              Generating...
            {:else}
              ✨ Generate Architecture
            {/if}
          </Button>
          <Button variant="secondary" onclick={() => {
            requirements = {};
            saveRequirements();
          }}>
            Reset
          </Button>
        </div>
      </Card>
    {/if}

    <!-- Clarifying Questions -->
    {#if clarifyingQuestions.length > 0}
      <Card class="mb-8">
        <h2 class="text-2xl font-semibold mb-4">🤔 Clarifying Questions</h2>
        <p class="text-gray-600 mb-4">Our AI needs more information to optimize your architecture:</p>
        <ul class="space-y-3 mb-6">
          {#each clarifyingQuestions as question, i}
            <li class="flex gap-3">
              <span class="font-semibold text-blue-600">{i + 1}.</span>
              <span>{question}</span>
            </li>
          {/each}
        </ul>
        <Button variant="primary" onclick={generateDiagram}>
          Answer Later & Generate
        </Button>
      </Card>
    {/if}

    <!-- Generated Diagram -->
    {#if mermaidCode}
      <Card class="mb-8">
        <h2 class="text-2xl font-semibold mb-4">📊 System Architecture Diagram</h2>
        <div id="mermaid-container" class="bg-white p-8 rounded-lg overflow-x-auto">
          <!-- Mermaid diagram renders here -->
        </div>
        <div class="mt-4 flex gap-4">
          <Button variant="secondary" onclick={() => navigator.clipboard.writeText(mermaidCode)}>
            📋 Copy Mermaid Code
          </Button>
          <Button variant="secondary" onclick={generateDiagram}>
            🔄 Regenerate
          </Button>
        </div>
      </Card>
    {/if}

    <!-- AI Recommendations -->
    {#if aiRecommendations.length > 0}
      <Card class="mb-8">
        <h2 class="text-2xl font-semibold mb-4">💡 AI Recommendations</h2>
        <ul class="space-y-3">
          {#each aiRecommendations as rec}
            <li class="flex gap-3">
              <span class="text-green-500">✓</span>
              <span>{rec}</span>
            </li>
          {/each}
        </ul>
      </Card>
    {/if}

    <!-- Cost Estimate -->
    {#if costEstimate}
      <Card>
        <h2 class="text-2xl font-semibold mb-4">💰 Estimated Monthly Cost</h2>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div class="text-center">
            <div class="text-3xl font-bold text-blue-600">${costEstimate.total}/mo</div>
            <div class="text-sm text-gray-600">Total</div>
          </div>
          <div>
            <h3 class="font-semibold mb-2">Breakdown:</h3>
            <ul class="space-y-1 text-sm">
              {#each Object.entries(costEstimate.breakdown || {}) as [service, cost]}
                <li class="flex justify-between">
                  <span>{service}</span>
                  <span class="font-mono">${cost}</span>
                </li>
              {/each}
            </ul>
          </div>
          <div>
            <h3 class="font-semibold mb-2">Optimization Tips:</h3>
            <ul class="space-y-1 text-sm text-gray-600">
              {#each costEstimate.optimization_tips || [] as tip}
                <li>• {tip}</li>
              {/each}
            </ul>
          </div>
        </div>
      </Card>
    {/if}
  </main>

  <Footer />
</div>
