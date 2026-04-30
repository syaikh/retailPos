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

<div class="bg-slate-800/50 backdrop-blur-sm border border-slate-700 rounded-xl p-6 {className}" {...rest}>
  {@render children()}
</div>
