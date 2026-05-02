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

<div class="stat-card">
  <div class="flex items-start justify-between">
    <div class="stat-card-icon {iconBg}">
      <Icon size={20} class={iconColor} />
    </div>

    {#if trend !== null}
      <span class="text-xs font-semibold px-2 py-0.5 rounded-full {trendUp ? 'badge-success' : trendDown ? 'badge-danger' : 'badge-muted'}">
        {trendUp ? '↑' : trendDown ? '↓' : '—'} {Math.abs(trend)}%
      </span>
    {/if}
  </div>

  <div>
    {#if loading}
      <div class="skeleton h-7 w-32 mb-1"></div>
      <div class="skeleton h-4 w-20"></div>
    {:else}
      <p class="text-2xl font-bold text-text-primary leading-none">{value}</p>
      {#if sub}
        <p class="text-xs text-text-muted mt-1">{sub}</p>
      {/if}
    {/if}
    <p class="text-xs text-text-muted mt-2 font-medium uppercase tracking-wide">{label}</p>
  </div>
</div>
