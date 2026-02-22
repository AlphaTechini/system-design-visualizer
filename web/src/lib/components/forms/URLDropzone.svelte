<script lang="ts">
	import { Upload, Link } from 'lucide-svelte';
	
	interface Props {
		urls?: string[];
	}

	let { urls = [] }: Props = $props();
	const dispatch = createEventDispatcher<{
		addUrl: { url: string };
		removeUrl: { index: number };
	}>();
	
	let isDragging = $state(false);
	let newUrl = $state('');
	
	function handleDrop(e: DragEvent) {
		e.preventDefault();
		isDragging = false;
		
		const text = e.dataTransfer?.getData('text');
		if (text && isValidUrl(text)) {
			dispatch('addUrl', { url: text });
		}
	}
	
	function handleAdd() {
		if (newUrl.trim() && isValidUrl(newUrl.trim())) {
			dispatch('addUrl', { url: newUrl.trim() });
			newUrl = '';
		}
	}
	
	function isValidUrl(string: string): boolean {
		try {
			new URL(string);
			return true;
		} catch (_) {
			return false;
		}
	}
</script>

<div class="space-y-4">
	<div
		class="dropzone"
		class:dropzone-active={isDragging}
		role="button"
		tabindex="0"
		aria-label="Drag and drop URLs here or paste below"
		ondragover={(e) => { e.preventDefault(); isDragging = true; }}
		ondragleave={() => isDragging = false}
		ondrop={handleDrop}
	>
		<div class="text-gray-500 mb-4">
			<Upload class="w-12 h-12 mx-auto mb-2" />
			<p class="font-medium">Drag and drop URLs here</p>
			<p class="text-sm mt-1">or paste below to add endpoints</p>
		</div>
		
		<div class="flex gap-2 max-w-xl mx-auto">
			<input
				type="url"
				bind:value={newUrl}
				placeholder="https://api.example.com/graphql"
				class="input flex-1"
				onkeydown={(e) => e.key === 'Enter' && handleAdd()}
			/>
			<button class="btn-primary" onclick={handleAdd}>
				Add Endpoint
			</button>
		</div>
	</div>
	
	{#if urls.length > 0}
		<div class="space-y-2">
			<h4 class="font-medium text-gray-700">Added Endpoints ({urls.length})</h4>
			{#each urls as url, index}
				<div class="flex items-center gap-2 bg-gray-50 rounded-md p-3">
					<Link class="w-5 h-5 text-gray-400 flex-shrink-0" />
					<span class="text-sm text-gray-700 font-mono flex-1 truncate">{url}</span>
					<button 
						onclick={() => dispatch('removeUrl', { index })}
						class="text-red-600 hover:text-red-800 text-sm font-medium"
						aria-label="Remove endpoint"
					>
						Remove
					</button>
				</div>
			{/each}
		</div>
	{/if}
</div>
