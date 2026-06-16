<script lang="ts">
  import { ChevronRight, ChevronDown } from 'lucide-svelte';
  import type { Snippet } from 'svelte';

  let {
    expanded = $bindable(false),
    rowContent,
    expandedContent,
    class: className = '',
  }: {
    expanded?: boolean;
    rowContent?: Snippet;
    expandedContent?: Snippet;
    class?: string;
  } = $props();

  function toggle() {
    expanded = !expanded;
  }
</script>

<tr class="border-t border-border/60 hover:bg-surface-hover/50 transition-colors group {className}" onclick={toggle} style="cursor: pointer;">
  <td class="w-10 px-2 py-3 text-center">
      <button
        type="button"
        class="p-1 rounded text-text-muted/70 hover:text-text-primary transition-all focus:outline-none inline-flex items-center justify-center"
        onclick={(e) => { e.stopPropagation(); toggle(); }}
        aria-expanded={expanded}
        aria-label="Toggle details"
      >
      {#if expanded}
        <ChevronDown size={16} />
      {:else}
        <ChevronRight size={16} />
      {/if}
    </button>
  </td>
  {@render rowContent?.()}
</tr>
{#if expanded}
  <tr class="bg-surface-default border-t border-border/40">
    <td colspan="100" class="p-0">
      <div class="animate-in slide-in-from-top-1 fade-in duration-200">
        <div class="px-10 py-5 border-l-2 border-primary-default bg-surface-subtle shadow-inner">
          {@render expandedContent?.()}
        </div>
      </div>
    </td>
  </tr>
{/if}
