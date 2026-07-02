<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		variant = 'default',
		size = 'md',
		class: className = '',
		children
	}: {
		variant?: 'default' | 'success' | 'warning' | 'danger';
		size?: 'sm' | 'md';
		class?: string;
		children: Snippet;
	} = $props();

	const variants = {
		default: 'bg-primary/20 text-primary',
		success: 'bg-success/20 text-success',
		warning: 'bg-warning/20 text-warning',
		danger: 'bg-danger/20 text-danger'
	};

	const sizes = {
		sm: 'px-2 py-0.5 text-xs',
		md: 'px-3 py-1 text-sm'
	};
</script>

<span class="badge {variants[variant]} {sizes[size]} {className}">
	{@render children()}
</span>

<style>
	.badge {
		display: inline-flex;
		align-items: center;
		border-radius: 9999px;
		font-weight: 600;
	}
</style>
