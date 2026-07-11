<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    title,
    subtitle = '',
    actions,
  }: {
    title: string;
    subtitle?: string;
    actions?: Snippet;
  } = $props();
</script>

<div class="flex items-center justify-between">
  <div>
    <h1 class="text-2xl font-bold text-text-primary">{title}</h1>
    {#if subtitle}
      <p class="mt-1 text-sm text-text-secondary">{subtitle}</p>
    {/if}
  </div>
  {#if actions}
    <div class="flex items-center gap-3 shrink-0">
      {@render actions()}
    </div>
  {/if}
</div>
