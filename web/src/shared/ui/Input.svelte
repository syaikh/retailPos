<script lang="ts">
  import { cn } from '$shared/utils/cn';
  import type { Snippet } from 'svelte';

  let {
    tag = 'input',
    class: className = '',
    value = $bindable(''),
    children,
    oninput: externalOninput,
    ...rest
  }: {
    tag?: 'input' | 'select' | 'textarea';
    class?: string;
    value?: string;
    children?: Snippet;
    oninput?: (e: Event) => void;
    [key: string]: any;
  } = $props();

  function handleInput(e: Event) {
    value = (e.target as HTMLInputElement | HTMLSelectElement).value;
    externalOninput?.(e);
  }
</script>

{#if tag === 'input'}
  <svelte:element
    this={tag}
    class={cn(
      'w-full rounded-xl border border-border-default bg-bg-secondary px-3.5 py-2.5 text-sm text-text-primary placeholder-text-muted focus:border-primary-default focus:outline-none focus:ring-2 focus:ring-primary-default/20 disabled:cursor-not-allowed disabled:opacity-40 transition-colors duration-200',
      className
    )}
    {value}
    oninput={handleInput}
    {...rest}
  />
{:else}
  <svelte:element
    this={tag}
    class={cn(
      'w-full rounded-xl border border-border-default bg-bg-secondary px-3.5 py-2.5 text-sm text-text-primary placeholder-text-muted focus:border-primary-default focus:outline-none focus:ring-2 focus:ring-primary-default/20 disabled:cursor-not-allowed disabled:opacity-40 transition-colors duration-200',
      className
    )}
    {value}
    oninput={handleInput}
    {...rest}
  >
    {#if tag === 'select'}
      {@render children?.()}
    {/if}
  </svelte:element>
{/if}
