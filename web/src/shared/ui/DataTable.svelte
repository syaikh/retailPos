<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    header,
    body,
    class: className = '',
    ariaLabel = '',
  }: {
    header?: Snippet;
    body?: Snippet;
    class?: string;
    ariaLabel?: string;
  } = $props();
</script>

<div class="overflow-x-auto w-full bg-surface-default border border-border rounded-xl shadow-sm">
  <table role="table" class="w-full text-sm text-left whitespace-nowrap {className}" aria-label={ariaLabel || undefined}>
    <thead class="sticky top-0 bg-surface-subtle z-10 border-b border-border">
      <tr class="text-[11px] font-semibold text-text-muted uppercase tracking-wider h-10">
        {@render header?.()}
      </tr>
    </thead>
    <tbody class="divide-y divide-border/60 bg-surface-default">
      {@render body?.()}
    </tbody>
  </table>
</div>
