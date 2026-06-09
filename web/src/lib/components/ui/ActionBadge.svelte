<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    action = '',
    class: className = '',
    children,
  }: {
    action: string;
    class?: string;
    children?: Snippet;
  } = $props();

  function getBadgeClass(action: string) {
    const v = (action || '').toUpperCase();
    if (v === 'CREATE') return 'bg-success-subtle text-success-light border border-success-default/20';
    if (v === 'UPDATE') return 'bg-warning-subtle text-warning-light border border-warning-default/20';
    if (v === 'DELETE') return 'bg-danger-subtle text-danger-light border border-danger-default/20';
    if (v === 'LOGIN') return 'bg-primary-subtle text-primary-light border border-primary-default/20';
    if (v === 'LOGOUT') return 'bg-surface-hover text-text-secondary border border-border-strong';
    return 'bg-surface-default text-text-muted border border-border';
  }
</script>

<span
  class="inline-flex items-center justify-center px-2.5 h-6 rounded-full text-sm font-medium leading-none {getBadgeClass(action)} {className}"
>
  {#if children}
    {@render children()}
  {:else}
    {action || '—'}
  {/if}
</span>

