<script lang="ts">
	interface Props {
		title?: string;
		subtitle?: string;
		padding?: string;
		className?: string;
	}

	let {
		title = undefined,
		subtitle = undefined,
		padding = 'p-6',
		className = ''
	}: Props = $props();
</script>

<div class="bg-white rounded-lg shadow-md {padding} {className}">
	{#if title || subtitle}
		<div class="mb-4">
			{#if title}
				<h3 class="text-lg font-semibold text-gray-900">{title}</h3>
			{/if}
			{#if subtitle}
				<p class="text-sm text-gray-600 mt-1">{subtitle}</p>
			{/if}
		</div>
	{/if}
	<slot />
</div>
