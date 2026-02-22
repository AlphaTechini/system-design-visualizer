<script lang="ts">
	interface Props {
		variant?: 'primary' | 'secondary' | 'danger';
		disabled?: boolean;
		loading?: boolean;
		type?: 'button' | 'submit';
		size?: 'sm' | 'md' | 'lg';
		className?: string;
	}

	let {
		variant = 'primary',
		disabled = false,
		loading = false,
		type = 'button',
		size = 'md',
		className = ''
	}: Props = $props();

	const sizeClasses = {
		sm: 'py-1 px-3 text-sm',
		md: 'py-2 px-4',
		lg: 'py-3 px-6 text-lg'
	};

	function getBaseClass() {
		const base = 'font-semibold rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2';
		const variantClasses = {
			primary: 'bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-white',
			secondary: 'bg-gray-200 hover:bg-gray-300 text-gray-800',
			danger: 'bg-red-600 hover:bg-red-700 text-white'
		};
		return `${base} ${variantClasses[variant]} ${sizeClasses[size]} ${className}`;
	}
	
	let baseClass = $derived(getBaseClass());
</script>

<button 
	type={type}
	class={baseClass}
	disabled={disabled || loading}
>
	{#if loading}
		<svg class="animate-spin h-5 w-5" viewBox="0 0 24 24">
			<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
			<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
		</svg>
	{/if}
	<slot />
</button>
