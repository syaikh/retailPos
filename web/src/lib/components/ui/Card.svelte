<script lang="ts">
  import { cn } from '$lib/utils/cn';
  import type { Snippet } from 'svelte';

  let {
    variant = 'default',
    class: className = '',
    children,
    ...rest
  }: {
    variant?: 'default' | 'glass' | 'hover';
    class?: string;
    children: Snippet;
  } = $props();

  const variants: Record<string, string> = {
    default: 'bg-surface-default border border-border-default rounded-2xl shadow-card',
    glass: 'bg-surface-default/70 backdrop-blur-xl border border-border-default rounded-2xl shadow-card',
    hover: 'bg-surface-default border border-border-default rounded-2xl shadow-card transition-all duration-300 hover:border-border-strong hover:shadow-card-hover hover:-translate-y-0.5',
  };
</script>

<div class={cn(variants[variant], className)} {...rest}>
  {@render children()}
</div>
