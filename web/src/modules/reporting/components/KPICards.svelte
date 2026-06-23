<script>
  import { Skeleton } from '$shared/ui';
  import { TrendingUp, TrendingDown } from 'lucide-svelte';
  import { formatCurrencyShort, formatLargeNumber } from '$modules/reporting/lib/reporting-utils';

  let {
    loading = true,
    kpiData = { totalRevenue: 0, previousRevenue: 0, totalOrders: 0, previousOrders: 0, avgOrderValue: 0, previousAvgOrderValue: 0, revenuePerDay: 0, previousRevenuePerDay: 0, peakRevenueHour: null, previousPeakRevenue: null, peakRevenueMonth: null, previousPeakRevenueMonth: null, percentChange: 0, comparisonType: 'zero', isPartial: false, periodInfo: null },
    chartType = 'hourly',
    activePeriodType = 'realtime',
    peakChartValue = null,
    projectedRevenue = null,
    aovTrend = null,
    statCardLabels = { card4: '', card5: '', comparisonLabel: '' },
    comparisonDateRange = '',
  } = $props();
</script>

<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3 mb-6">
  {#if loading}
    {#each { length: 5 } as _}
      <div class="bg-surface rounded-lg p-3 border border-border/50 min-w-0">
        <Skeleton width="w-16" height="h-2.5" class="mb-1.5" />
        <Skeleton width="w-12" height="h-5" />
      </div>
    {/each}
  {:else}
    <div class="bg-surface rounded-lg p-4 border border-border/50">
      <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Total Revenue</div>
      <div class="text-lg font-bold text-text-primary mt-1">
        {formatCurrencyShort(kpiData.totalRevenue)}
      </div>
      {#if chartType === 'hourly' && peakChartValue !== null}
        <div class="text-xs text-text-muted mt-1">
          Peak: {formatCurrencyShort(peakChartValue)}
        </div>
      {/if}
      {#if projectedRevenue !== null}
        <div class="text-xs text-success mt-1 font-medium">
          Projected: {formatCurrencyShort(projectedRevenue)}
        </div>
      {/if}
      {#if kpiData.previousRevenue > 0}
        <div class="text-xs text-text-secondary mt-1 font-medium">
          vs {formatCurrencyShort(kpiData.previousRevenue)}
        </div>
      {/if}
    </div>

    <div class="bg-surface rounded-lg p-4 border border-border/50 relative">
      <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Total Orders</div>
      <div class="text-lg font-bold text-text-primary mt-1">
        {formatLargeNumber(kpiData.totalOrders)}
      </div>
      {#if kpiData.previousOrders > 0}
        <div class="text-xs text-text-secondary mt-1 font-medium">
          vs {formatLargeNumber(kpiData.previousOrders)}
        </div>
      {/if}
    </div>

    <div class="bg-surface rounded-lg p-4 border border-border/50">
      <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Avg Order Value</div>
      <div class="flex items-center gap-1 mt-1">
        <span class="text-lg font-bold text-text-primary">
          {formatCurrencyShort(kpiData.avgOrderValue)}
        </span>
        {#if aovTrend === 'up'}
          <TrendingUp size={14} class="text-success" />
        {:else if aovTrend === 'down'}
          <TrendingDown size={14} class="text-danger-light" />
        {/if}
      </div>
      {#if kpiData.previousAvgOrderValue > 0}
        <div class="text-xs text-text-secondary mt-1 font-medium">
          vs {formatCurrencyShort(kpiData.previousAvgOrderValue)}
        </div>
      {/if}
    </div>

    <div class="bg-surface rounded-lg p-4 border border-border/50">
      <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">
        {statCardLabels.card4}
      </div>
      <div class="flex items-baseline gap-1 mt-1">
        <span class="text-lg font-bold text-text-primary">
          {formatCurrencyShort(
            chartType === 'hourly' ? kpiData.peakRevenueHour :
            chartType === 'yearly' ? kpiData.peakRevenueMonth :
            kpiData.revenuePerDay
          )}
        </span>
      </div>
      {#if chartType === 'yearly' && kpiData.revenuePerDay > 0}
        <div class="text-xs text-text-muted mt-1">
          Avg. / Month: {formatCurrencyShort(kpiData.revenuePerDay * 30)}
        </div>
      {/if}
      {#if chartType === 'hourly' && kpiData.previousPeakRevenue !== null && kpiData.previousPeakRevenue > 0}
        <div class="text-xs text-text-secondary mt-1 font-medium">
          vs {formatCurrencyShort(kpiData.previousPeakRevenue)}
        </div>
      {:else if chartType === 'yearly' && kpiData.previousPeakRevenueMonth > 0}
        <div class="text-xs text-text-secondary mt-1 font-medium">
          vs {formatCurrencyShort(kpiData.previousPeakRevenueMonth)}
        </div>
      {:else if kpiData.previousRevenuePerDay > 0}
        <div class="text-xs text-text-secondary mt-1 font-medium">
          vs {formatCurrencyShort(kpiData.previousRevenuePerDay)}
        </div>
      {/if}
    </div>

    <div class="bg-surface rounded-lg p-4 border border-border/50">
      <div class="text-xs font-medium text-text-secondary uppercase tracking-wide flex items-center gap-1">
        {statCardLabels.comparisonLabel}
      </div>
      <div class="flex items-baseline gap-1 mt-1">
        {#if kpiData.percentChange !== null}
          <span class={`text-lg font-bold ${
            kpiData.comparisonType === 'new' ? 'text-success' :
            kpiData.comparisonType === 'zero' ? 'text-text-secondary' :
            kpiData.percentChange > 0 ? 'text-success' : 'text-danger'
          }`}>
            {kpiData.comparisonType === 'new' ? 'NEW' :
             kpiData.comparisonType === 'zero' ? '±0%' :
             kpiData.percentChange >= 0 ? '+' + kpiData.percentChange.toFixed(1) + '%' :
             kpiData.percentChange.toFixed(1) + '%'}
          </span>
          {#if kpiData.comparisonType !== 'new' && kpiData.comparisonType !== 'zero'}
            {#if kpiData.percentChange > 0}
              <TrendingUp size={14} class="text-success" />
            {:else}
              <TrendingDown size={14} class="text-danger-light" />
            {/if}
          {/if}
        {/if}
      </div>
    </div>
  {/if}
</div>
