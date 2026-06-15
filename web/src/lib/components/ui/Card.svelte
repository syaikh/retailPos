<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    children,
    class: className = '',
    ...rest
  }: {
    class?: string;
    children: Snippet;
  } = $props();
</script>

<div class="bg-surface-default border border-border-default rounded-2xl shadow-card p-6 {className}" {...rest}>
  {@render children()}
</div>
