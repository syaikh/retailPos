<script>
  import { TrendingUp, TrendingDown, ChevronDown } from 'lucide-svelte';
  import { formatCurrencyShort, getPeriodLabel } from '$modules/reporting/lib/reporting-utils';

  let {
    bestPeriod = null,
    worstPeriod = null,
    bestWorstHeading = 'Period',
    tableRows = [],
    showDataTable = $bindable(false),
  } = $props();
</script>

{#if bestPeriod || worstPeriod}
  <div class="flex flex-wrap items-center gap-3 mb-4 px-1">
    {#if bestPeriod}
      <div class="flex items-center gap-1.5 text-xs bg-success/10 text-success-light px-2.5 py-1.5 rounded-full border border-success/20">
        <TrendingUp size={12} />
        <span class="font-medium">Best {bestWorstHeading}:</span>
        <span>{getPeriodLabel(bestPeriod)}</span>
        <span class="font-semibold">{formatCurrencyShort(bestPeriod.total || 0)}</span>
      </div>
    {/if}
    {#if worstPeriod && worstPeriod.total !== bestPeriod?.total}
      <div class="flex items-center gap-1.5 text-xs bg-danger/10 text-danger-light px-2.5 py-1.5 rounded-full border border-danger/20">
        <TrendingDown size={12} />
        <span class="font-medium">Worst {bestWorstHeading}:</span>
        <span>{getPeriodLabel(worstPeriod)}</span>
        <span class="font-semibold">{formatCurrencyShort(worstPeriod.total || 0)}</span>
      </div>
    {/if}
    {#if tableRows.length > 0}
      <button
        class="text-xs text-text-muted hover:text-text-secondary transition-colors ml-auto flex items-center gap-1"
        onclick={() => showDataTable = !showDataTable}
      >
        {showDataTable ? 'Hide' : 'Show'} Data Table
        <ChevronDown size={12} class="transition-transform duration-200 {showDataTable ? 'rotate-180' : ''}" />
      </button>
    {/if}
  </div>
{/if}
