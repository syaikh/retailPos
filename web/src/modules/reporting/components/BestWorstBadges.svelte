<script>
  import { TrendingUp, TrendingDown, ChevronDown } from 'lucide-svelte';
  import { formatCurrencyShort, getPeriodLabel } from '$modules/reporting/lib/reporting-utils';
  import { labels } from '$shared/i18n';

  let {
    bestPeriod = null,
    worstPeriod = null,
    bestWorstHeading = 'Period',
    tableRows = [],
    showDataTable = $bindable(false),
    isHourly = false,
    noCurrentData = false,
  } = $props();
</script>

{#if tableRows.length > 0}
  <div class="flex flex-wrap items-center gap-3 mb-4 px-1">
    {#if bestPeriod}
      <div class="flex items-center gap-1.5 text-xs bg-success/10 text-success-light px-2.5 py-1.5 rounded-full border border-success/20">
        <TrendingUp size={12} />
        <span class="font-medium">{labels.best} {bestWorstHeading}:</span>
        <span>{getPeriodLabel(bestPeriod)}</span>
        <span class="font-semibold">{formatCurrencyShort(bestPeriod.total || 0)}</span>
      </div>
    {:else if noCurrentData}
      <div class="flex items-center gap-1.5 text-xs bg-surface text-text-muted px-2.5 py-1.5 rounded-full border border-border">
        <TrendingUp size={12} />
        <span class="font-medium">{labels.best} {bestWorstHeading}:</span>
        <span>-</span>
        <span class="font-semibold">-</span>
      </div>
    {/if}
    {#if worstPeriod && worstPeriod.total !== bestPeriod?.total}
      <div class="flex items-center gap-1.5 text-xs bg-danger/10 text-danger-light px-2.5 py-1.5 rounded-full border border-danger/20" title={labels.zeroRevenueHoursExcluded}>
        <TrendingDown size={12} />
        <span class="font-medium">{labels.worst} {bestWorstHeading}:</span>
        <span>{getPeriodLabel(worstPeriod)}</span>
        <span class="font-semibold">{formatCurrencyShort(worstPeriod.total || 0)}</span>
        {#if isHourly}
          <span class="text-text-muted/60 italic ml-1">{labels.exclZeroRevenueHours}</span>
        {/if}
      </div>
    {:else if noCurrentData && !bestPeriod}
      <div class="flex items-center gap-1.5 text-xs bg-surface text-text-muted px-2.5 py-1.5 rounded-full border border-border">
        <TrendingDown size={12} />
        <span class="font-medium">{labels.worst} {bestWorstHeading}:</span>
        <span>-</span>
        <span class="font-semibold">-</span>
      </div>
    {/if}
    {#if tableRows.length > 0}
      <button
        class="text-xs text-text-muted hover:text-text-secondary transition-colors ml-auto flex items-center gap-1"
        onclick={() => showDataTable = !showDataTable}
      >
        {showDataTable ? labels.hide : labels.show} Data Table
        <ChevronDown size={12} class="transition-transform duration-200 {showDataTable ? 'rotate-180' : ''}" />
      </button>
    {/if}
  </div>
{/if}
