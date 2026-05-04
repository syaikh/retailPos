<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { Component } from 'svelte';

  let {
    label,
    value,
    sub = '',
    trend = null,
    icon: Icon,
    iconBg = 'bg-primary-subtle',
    iconColor = 'text-primary-light',
    loading = false,
  }: {
    label: string;
    value: string | number;
    sub?: string;
    trend?: number | null;     // positive = up, negative = down, null = no trend
    icon: Component;
    iconBg?: string;
    iconColor?: string;
    loading?: boolean;
  } = $props();

  const trendUp = $derived(trend !== null && trend > 0);
  const trendDown = $derived(trend !== null && trend < 0);
</script>

<div class="flex items-center gap-4">
  <div class="w-14 h-14 rounded-xl {iconBg} flex items-center justify-center flex-shrink-0">
    <Icon size={28} class={iconColor} />
  </div>

  <div class="flex-1 text-left">
    {#if loading}
      <div class="skeleton h-10 w-48 mb-2"></div>
      <div class="skeleton h-6 w-32"></div>
      <div class="skeleton h-4 w-24"></div>
    {:else}
      <p class="text-4xl font-bold text-text-primary leading-none">{value}</p>
      {#if sub}
        <p class="text-base text-text-muted mt-1">{sub}</p>
      {/if}
      <p class="text-sm text-text-muted mt-2 font-medium uppercase tracking-wide">{label}</p>
    {/if}

    {#if trend !== null}
      <span class="inline-block text-sm font-semibold px-2.5 py-1 rounded-full mt-2 {trendUp ? 'badge-success' : trendDown ? 'badge-danger' : 'badge-muted'}">
        {trendUp ? '↑' : trendDown ? '↓' : '—'} {Math.abs(trend)}%
      </span>
    {/if}
  </div>
</div>
