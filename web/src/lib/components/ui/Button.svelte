<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	let {
		variant = 'primary',
		size = 'md',
		disabled = false,
		loading = false,
		class: className = '',
		children,
		...rest
	}: HTMLButtonAttributes & {
		variant?: 'primary' | 'secondary' | 'ghost' | 'destructive';
		size?: 'sm' | 'md' | 'lg';
		loading?: boolean;
		class?: string;
		children: Snippet;
	} = $props();

	const variants = {
		primary: 'bg-primary text-white hover:bg-primary-dark disabled:opacity-50',
		secondary: 'bg-bg-main text-text-primary border border-border hover:bg-opacity-80',
		ghost: 'bg-transparent text-text-primary hover:bg-white/10',
		destructive: 'bg-danger text-white hover:bg-red-600'
	};

	const sizes = {
		sm: 'px-2 py-1 text-sm rounded',
		md: 'px-4 py-2 text-base rounded',
		lg: 'px-6 py-3 text-lg rounded'
	};
</script>

<button
	class="btn {variants[variant]} {sizes[size]} {className}"
	{disabled}
	{...rest}
>
	{#if loading}
		<svg class="spinner" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
			<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" fill="none" stroke-dasharray="30 60" />
		</svg>
	{/if}
	{@render children()}
</button>

<style>
	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.2s;
		border: none;
	}

	.btn:disabled {
		cursor: not-allowed;
	}

	.spinner {
		width: 16px;
		height: 16px;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}
</style>
