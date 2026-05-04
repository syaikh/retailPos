<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    variant = 'default',
    size = 'md',
    class: className = '',
    children
  }: {
    variant?: 'default' | 'success' | 'warning' | 'danger' | 'destructive' | 'primary' | 'muted';
    size?: 'sm' | 'md';
    class?: string;
    children: Snippet;
  } = $props();

  const variants = {
    default: 'bg-blue-500/20 text-blue-300',
    success: 'bg-emerald-500/20 text-emerald-300',
    warning: 'bg-amber-500/20 text-amber-300',
    danger: 'bg-red-500/20 text-red-300',
    destructive: 'bg-red-500/20 text-red-300',
    primary: 'bg-primary-subtle text-primary-light',
    muted: 'bg-surface-default text-text-muted border border-border-default'
  };

  const sizes = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-3 py-1 text-sm'
  };
</script>

<span class="{variants[variant]} {sizes[size]} {className} rounded-full font-medium">
  {@render children()}
</span>
