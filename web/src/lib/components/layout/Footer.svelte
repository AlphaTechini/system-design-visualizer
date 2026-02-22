<script lang="ts">
	const currentYear = new Date().getFullYear();
</script>

<footer class="bg-white border-t mt-auto">
	<div class="max-w-7xl mx-auto px-4 py-6">
		<div class="flex flex-col md:flex-row justify-between items-center gap-4">
			<div class="text-sm text-gray-600">
				© {currentYear} System Design Visualizer. Built with ❤️ using SvelteKit + Go
			</div>
			<div class="flex items-center gap-6 text-sm text-gray-600">
				<a href="https://github.com/AlphaTechini/system-design-visualizer" target="_blank" class="hover:text-[var(--primary)] transition-colors">
					GitHub
				</a>
				<span class="text-gray-400">Documentation (coming soon)</span>
				<span class="text-gray-400">API Reference (coming soon)</span>
			</div>
		</div>
	</div>
</footer>
