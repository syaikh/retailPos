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

<div class="page-header">
  <div>
    <h1 class="page-title">{title}</h1>
    {#if subtitle}
      <p class="page-subtitle">{subtitle}</p>
    {/if}
  </div>
  {#if actions}
    <div class="flex items-center gap-3 shrink-0">
      {@render actions()}
    </div>
  {/if}
</div>
