<script>
  import { fly } from 'svelte/transition';
  import { TrendingUp, TrendingDown } from 'lucide-svelte';

  let {
    showDataTable = false,
    sortedRows = [],
    sortColumn = $bindable('period'),
    sortAsc = $bindable(true),
    tablePeriodHeading = 'Period',
    ontogglesort = (col) => {},
  } = $props();
</script>

{#if showDataTable && sortedRows.length > 0}
  <div class="mt-5 overflow-x-auto" transition:fly={{ y: -8, duration: 200 }}>
    <table class="w-full text-xs text-left border-collapse">
      <thead>
        <tr class="border-b border-border/50">
          <th class="py-2 px-3 font-medium text-text-secondary select-none whitespace-nowrap">
            <button type="button" class="flex items-center gap-1 hover:text-text-primary transition-colors" onclick={() => ontogglesort('period')}>
              {tablePeriodHeading}
              {#if sortColumn === 'period'}
                <span>{sortAsc ? '▲' : '▼'}</span>
              {/if}
            </button>
          </th>
          <th class="py-2 px-3 font-medium text-text-secondary !text-right select-none whitespace-nowrap">
            <button type="button" class="flex items-center gap-1 hover:text-text-primary transition-colors justify-end w-full" onclick={() => ontogglesort('revenue')}>
              Revenue (Rp)
              {#if sortColumn === 'revenue'}
                <span>{sortAsc ? '▲' : '▼'}</span>
              {/if}
            </button>
          </th>
          <th class="py-2 px-3 font-medium text-text-secondary !text-right select-none whitespace-nowrap">
            <button type="button" class="flex items-center gap-1 hover:text-text-primary transition-colors justify-end w-full" onclick={() => ontogglesort('prev')}>
              Prev Period (Rp)
              {#if sortColumn === 'prev'}
                <span>{sortAsc ? '▲' : '▼'}</span>
              {/if}
            </button>
          </th>
          <th class="py-2 px-3 font-medium text-text-secondary !text-right select-none whitespace-nowrap">
            <button type="button" class="flex items-center gap-1 hover:text-text-primary transition-colors justify-end w-full" onclick={() => ontogglesort('change')}>
              Change
              {#if sortColumn === 'change'}
                <span>{sortAsc ? '▲' : '▼'}</span>
              {/if}
            </button>
          </th>
          {#if sortedRows.some(r => r.orderCount !== null)}
            <th class="py-2 px-3 font-medium text-text-secondary text-right whitespace-nowrap">
              Orders
            </th>
          {/if}
        </tr>
      </thead>
      <tbody>
        {#each sortedRows as row (row.dateStr || row.period)}
          {@const change = row.prevRevenue > 0 ? ((row.revenue - row.prevRevenue) / row.prevRevenue) * 100 : null}
          <tr class="border-b border-border/30 hover:bg-surface-hover/50 transition-colors">
            <td class="py-2 px-3 text-text-primary whitespace-nowrap">{row.period}</td>
            <td class="py-2 px-3 text-text-primary text-right font-medium whitespace-nowrap">{row.revenue.toLocaleString('id-ID')}</td>
            <td class="py-2 px-3 text-text-secondary text-right whitespace-nowrap">
              {#if row.prevRevenue !== null}
                {row.prevRevenue.toLocaleString('id-ID')}
              {:else}
                <span class="text-text-muted">—</span>
              {/if}
            </td>
            <td class="py-2 px-3 text-right whitespace-nowrap">
              {#if change !== null}
                <span class:font-medium={true} class:text-success={change > 0} class:text-danger={change < 0} class:text-text-muted={change === 0}>
                  {change > 0 ? '+' : ''}{change.toFixed(1)}%
                </span>
                {#if change > 0}
                  <TrendingUp size={10} class="inline text-success ml-0.5" />
                {:else if change < 0}
                  <TrendingDown size={10} class="inline text-danger ml-0.5" />
                {/if}
              {:else}
                <span class="text-text-muted">—</span>
              {/if}
            </td>
            {#if sortedRows.some(r => r.orderCount !== null)}
              <td class="py-2 px-3 text-text-secondary text-right whitespace-nowrap">{row.orderCount ?? '—'}</td>
            {/if}
          </tr>
        {/each}
      </tbody>
      <tfoot>
        {#if true}
          {@const totalRevenue = sortedRows.reduce((s, r) => s + (r.revenue || 0), 0)}
          {@const totalPrev = sortedRows.reduce((s, r) => s + (r.prevRevenue || 0), 0)}
          {@const hasPrevOverall = sortedRows.some(r => r.prevRevenue !== null)}
          <tr class="border-t-2 border-border/60 font-semibold">
            <td class="py-2.5 px-3 text-text-primary text-sm">Total</td>
            <td class="py-2.5 px-3 text-text-primary text-right text-sm">{totalRevenue.toLocaleString('id-ID')}</td>
            <td class="py-2.5 px-3 text-text-secondary text-right text-sm">
              {#if hasPrevOverall}
                {totalPrev.toLocaleString('id-ID')}
              {:else}
                <span class="text-text-muted">—</span>
              {/if}
            </td>
            <td class="py-2.5 px-3 text-right text-sm">
              {#if hasPrevOverall && totalPrev > 0}
                {@const totalChange = ((totalRevenue - totalPrev) / totalPrev) * 100}
                <span class:font-bold={true} class:text-success={totalChange > 0} class:text-danger={totalChange < 0}>
                  {totalChange > 0 ? '+' : ''}{totalChange.toFixed(1)}%
                </span>
              {:else}
                <span class="text-text-muted">—</span>
              {/if}
            </td>
            {#if sortedRows.some(r => r.orderCount !== null)}
              {@const totalOrders = sortedRows.reduce((s, r) => s + (r.orderCount || 0), 0)}
              <td class="py-2.5 px-3 text-text-secondary text-right text-sm">{totalOrders}</td>
            {/if}
          </tr>
        {/if}
      </tfoot>
    </table>
  </div>
{/if}
