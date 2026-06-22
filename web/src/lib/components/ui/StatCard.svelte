<script>
  import Badge from '$lib/components/ui/Badge.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';

  let {
    label,
    value,
    sub = '',
    trend = null,
    trendLabel = 'vs kemarin',
    displayTrend = false,
    icon: Icon,
    iconBg = 'bg-primary-subtle',
    iconColor = 'text-primary-light',
    loading = false,
    valueClass = '',
  } = $props();

  const trendUp = $derived(trend !== null && trend > 0);
  const trendDown = $derived(trend !== null && trend < 0);
</script>

<div class="card card-hover p-6 rounded-2xl border border-border relative overflow-hidden transition-all duration-300 hover:shadow-lg">
  <div class="absolute inset-x-0 top-0 h-0.5 bg-linear-to-r from-transparent via-border to-transparent"></div>
  <div class="flex items-center gap-4">
    <div class="w-14 h-14 rounded-xl {iconBg} flex items-center justify-center shrink-0 ring-1 ring-black/5">
      <Icon size={28} class={iconColor} />
    </div>

    <div class="flex-1 min-w-0 text-left">
      {#if loading}
        <Skeleton width="w-56" height="h-12" class="mb-2" />
        <Skeleton width="w-44" height="h-6" />
      {:else}
        <p class="text-4xl font-bold text-text-primary leading-none truncate transition-all duration-300 {valueClass}">{value}</p>
        {#if displayTrend && trend !== null}
          <p class="flex items-center gap-1.5 text-sm font-semibold mt-1.5 {trendUp ? 'text-success-light' : trendDown ? 'text-danger-light' : 'text-text-muted'}">
            {trendUp ? '↑' : trendDown ? '↓' : '—'} {trend ? Math.abs(trend) : 0}% {trendLabel}
          </p>
        {:else if sub}
          <p class="text-base text-text-muted font-medium mt-1.5 truncate">{sub}</p>
        {/if}
        <p class="text-sm text-text-muted mt-3 font-semibold uppercase tracking-widest">{label}</p>
      {/if}

      {#if trend !== null && !displayTrend}
        <Badge variant={trendUp ? 'success' : trendDown ? 'danger' : 'muted'} size="md" class="mt-2 border">
          {trendUp ? '↑' : trendDown ? '↓' : '—'} {trend ? Math.abs(trend) : 0}%
        </Badge>
      {/if}
    </div>
  </div>
</div>
