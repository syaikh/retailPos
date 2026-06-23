<script lang="ts">
  import { cn } from '$shared/utils/cn';
  import type { Snippet } from 'svelte';

  let {
    variant = 'primary',
    size = 'md',
    disabled = false,
    class: className = '',
    children,
    ...rest
  }: {
    variant?: 'primary' | 'secondary' | 'success' | 'danger' | 'ghost';
    size?: 'sm' | 'md' | 'icon' | 'xs';
    disabled?: boolean;
    class?: string;
    children: Snippet;
  } = $props();

  const base =
    'inline-flex items-center justify-center gap-2 rounded-xl text-sm font-semibold transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-bg-default disabled:opacity-40 disabled:cursor-not-allowed select-none';

  const variants: Record<string, string> = {
    primary:
      'bg-primary-default text-white hover:bg-primary-hover focus:ring-primary-default shadow-glow-primary-sm hover:shadow-glow-primary active:scale-[0.97]',
    secondary:
      'bg-surface-default text-text-primary border border-border-strong hover:bg-surface-hover focus:ring-primary-default active:scale-[0.97]',
    success:
      'bg-success-default text-white hover:bg-emerald-600 focus:ring-success-default shadow-glow-success active:scale-[0.97]',
    danger:
      'bg-danger-default text-white hover:bg-rose-600 focus:ring-danger-default shadow-glow-danger active:scale-[0.97]',
    ghost:
      'bg-transparent text-text-secondary hover:text-text-primary hover:bg-surface-default focus:ring-primary-default active:scale-[0.97]',
  };

  const sizes: Record<string, string> = {
    sm: 'px-3 py-1.5 text-xs rounded-lg',
    md: 'px-4 py-2',
    icon: 'p-2 rounded-lg aspect-square',
    xs: 'px-2 py-1 text-xs rounded-lg',
  };
</script>

<button
  type="button"
  class={cn(base, variants[variant], sizes[size], className)}
  {disabled}
  {...rest}
>
  {@render children()}
</button>
