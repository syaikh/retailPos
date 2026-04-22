<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		children,
		class: className = '',
		...rest
	}: HTMLAttributes<HTMLDivElement> & {
		class?: string;
		children: Snippet;
	} = $props();
</script>

<div class="card {className}" {...rest}>
	{@render children()}
</div>

<style>
	.card {
		background: var(--color-bg-elevated);
		border: 1px solid var(--color-border);
		border-radius: 12px;
		padding: 24px;
	}
</style>
