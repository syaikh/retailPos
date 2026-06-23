<script lang="ts">
  import { cn } from '$shared/utils/cn';
  import type { Snippet } from 'svelte';

  let {
    tag = 'input',
    class: className = '',
    children,
    ...rest
  }: {
    tag?: 'input' | 'select' | 'textarea';
    class?: string;
    children?: Snippet;
    [key: string]: any;
  } = $props();
</script>

<svelte:element
  this={tag}
  class={cn(
    'w-full rounded-xl border border-border-default bg-bg-secondary px-3.5 py-2.5 text-sm text-text-primary placeholder-text-muted focus:border-primary-default focus:outline-none focus:ring-2 focus:ring-primary-default/20 disabled:cursor-not-allowed disabled:opacity-40 transition-colors duration-200',
    className
  )}
  {...rest}
>
  {#if tag === 'select'}
    {@render children?.()}
  {/if}
</svelte:element>
