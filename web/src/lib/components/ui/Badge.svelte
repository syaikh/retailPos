<script lang="ts">
  import { cn } from '$lib/utils/cn';
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

  const base = 'inline-flex items-center gap-1.5 rounded-full font-semibold';

  const variants: Record<string, string> = {
    default: 'bg-info-subtle text-info-light shadow-[0_0_12px_rgba(14,165,233,0.15)]',
    success: 'bg-success-subtle text-success-light shadow-[0_0_12px_rgba(16,185,129,0.15)]',
    warning: 'bg-warning-subtle text-warning-light shadow-[0_0_12px_rgba(245,158,11,0.15)]',
    danger: 'bg-danger-subtle text-danger-light shadow-[0_0_12px_rgba(239,68,68,0.15)]',
    destructive: 'bg-danger-subtle text-danger-light shadow-[0_0_12px_rgba(239,68,68,0.15)]',
    primary: 'bg-primary-subtle text-primary-light shadow-glow-primary-sm',
    muted: 'bg-surface-default text-text-muted border border-border-default'
  };

  const sizes: Record<string, string> = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-3 py-1 text-sm'
  };
</script>

<span class={cn(base, variants[variant], sizes[size], className)}>
  {@render children()}
</span>
