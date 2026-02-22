<script lang="ts">
	import { Home, FileText, Settings, Github } from 'lucide-svelte';
</script>

<header class="bg-white shadow-sm border-b sticky top-0 z-50">
	<div class="max-w-7xl mx-auto px-4 py-4 flex justify-between items-center">
		<div class="flex items-center gap-3">
			<div class="w-10 h-10 bg-[var(--primary)] rounded-lg flex items-center justify-center">
				<span class="text-white text-xl">🏗️</span>
			</div>
			<div>
				<h1 class="text-xl font-bold text-gray-900">System Design Visualizer</h1>
				<p class="text-xs text-gray-500">AI-Powered Architecture Designer</p>
			</div>
		</div>
		
		<nav class="flex items-center gap-6">
			<a href="/" class="flex items-center gap-2 text-gray-600 hover:text-[var(--primary)] transition-colors">
				<Home class="w-5 h-5" />
				<span class="hidden md:inline">Dashboard</span>
			</a>
			<a href="/designs" class="flex items-center gap-2 text-gray-600 hover:text-[var(--primary)] transition-colors">
				<FileText class="w-5 h-5" />
				<span class="hidden md:inline">Designs</span>
			</a>
			<a href="/settings" class="flex items-center gap-2 text-gray-600 hover:text-[var(--primary)] transition-colors">
				<Settings class="w-5 h-5" />
				<span class="hidden md:inline">Settings</span>
			</a>
			<a 
				href="https://github.com/AlphaTechini/system-design-visualizer" 
				target="_blank"
				class="flex items-center gap-2 text-gray-600 hover:text-[var(--primary)] transition-colors"
			>
				<Github class="w-5 h-5" />
				<span class="hidden md:inline">GitHub</span>
			</a>
		</nav>
	</div>
</header>
