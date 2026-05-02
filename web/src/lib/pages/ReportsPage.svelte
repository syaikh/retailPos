<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import PageHeader from '$lib/components/ui/PageHeader.svelte';
  import StatCard from '$lib/components/ui/StatCard.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import {
    DollarSign, Receipt, TrendingUp, BarChart3,
    CalendarDays, Download, FileSpreadsheet,
  } from 'lucide-svelte';

  let loading = $state(true);
  let salesData = $state([]);

  // Date range
  let startDate = $state(new Date(new Date().setDate(1)).toISOString().slice(0, 10));
  let endDate = $state(new Date().toISOString().slice(0, 10));

  const stats = $derived({
    revenue: salesData.reduce((s, r) => s + (r.total_amount || 0), 0),
    orders:  salesData.length,
    avgOrder: salesData.length
      ? Math.floor(salesData.reduce((s, r) => s + (r.total_amount || 0), 0) / salesData.length)
      : 0,
  });

  async function fetchSales() {
    try {
      loading = true;
      const r = await apiFetch(`/api/sales?startDate=${startDate}&endDate=${endDate}`);
      if (r.ok) {
        const data = await r.json();
        salesData = data.data || [];
      }
    } catch {
      toast.error('Failed to load sales data');
    } finally {
      loading = false;
    }
  }

  const statusVariant = (s) =>
    s === 'completed' ? 'success' : s === 'refunded' ? 'danger' : 'warning';

  onMount(fetchSales);
</script>

<div class="space-y-5">
  <PageHeader title="Sales Reports" subtitle="Analytics and business reporting">
    {#snippet actions()}
      <button class="btn btn-secondary">
        <FileSpreadsheet size={15} /> Export Excel
      </button>
      <button class="btn btn-primary">
        <Download size={15} /> Export PDF
      </button>
    {/snippet}
  </PageHeader>

  <!-- Date range filter -->
  <div class="card p-4 flex flex-wrap items-center gap-4">
    <div class="flex items-center gap-2 text-sm font-medium text-text-secondary">
      <CalendarDays size={16} class="text-text-muted" />
      Date Range
    </div>
    <div class="flex items-center gap-3">
      <input type="date" class="input w-40" bind:value={startDate} max={endDate} />
      <span class="text-text-muted text-sm">to</span>
      <input type="date" class="input w-40" bind:value={endDate} min={startDate} max={new Date().toISOString().slice(0,10)} />
    </div>
    <button class="btn btn-primary btn-sm" onclick={fetchSales}>Apply</button>
  </div>

  <!-- KPI stats -->
  <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
    <StatCard
      label="Total Revenue"
      value={loading ? '—' : `Rp ${stats.revenue.toLocaleString('id-ID')}`}
      {loading}
      icon={DollarSign}
      iconBg="bg-primary-subtle"
      iconColor="text-primary-light"
    />
    <StatCard
      label="Total Orders"
      value={loading ? '—' : stats.orders}
      {loading}
      icon={Receipt}
      iconBg="bg-success-subtle"
      iconColor="text-success-light"
    />
    <StatCard
      label="Avg Order Value"
      value={loading ? '—' : `Rp ${stats.avgOrder.toLocaleString('id-ID')}`}
      {loading}
      icon={TrendingUp}
      iconBg="bg-info-subtle"
      iconColor="text-info-light"
    />
  </div>

  <!-- Chart placeholder -->
  <div class="card p-5">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-sm font-semibold text-text-primary">Revenue Overview</h3>
      <span class="badge badge-muted">This period</span>
    </div>
    <div class="flex items-center justify-center h-48 rounded-xl bg-bg border border-border border-dashed">
      <div class="text-center">
        <BarChart3 size={36} class="text-text-muted mx-auto mb-2 opacity-40" />
        <p class="text-text-muted text-sm">Chart visualization</p>
        <p class="text-text-muted text-xs mt-1">Connect Chart.js to render data</p>
      </div>
    </div>
  </div>

  <!-- Sales table -->
  <div class="card p-0 overflow-hidden">
    <div class="px-4 py-3 border-b border-border flex items-center justify-between">
      <p class="text-sm font-semibold text-text-primary">Transaction History</p>
      {#if !loading}
        <span class="badge badge-muted">{salesData.length} records</span>
      {/if}
    </div>

    {#if loading}
      <div class="divide-y divide-border">
        {#each { length: 5 } as _}
          <div class="flex items-center gap-4 px-4 py-3.5">
            <Skeleton width="w-32" height="h-4" />
            <Skeleton width="w-24" height="h-4" />
            <Skeleton width="w-20" height="h-6" rounded="rounded-full" class="ml-auto" />
            <Skeleton width="w-28" height="h-4" />
          </div>
        {/each}
      </div>
    {:else if salesData.length === 0}
      <div class="empty-state py-16">
        <div class="empty-state-icon bg-surface w-20 h-20">
          <Receipt size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold">No transactions found</p>
        <p class="text-text-muted text-sm mt-1">Try adjusting the date range</p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table>
          <thead>
            <tr>
              <th>Invoice</th>
              <th>Date</th>
              <th>Items</th>
              <th>Payment</th>
              <th>Status</th>
              <th class="text-right">Total</th>
            </tr>
          </thead>
          <tbody>
            {#each salesData as sale (sale.id)}
              <tr>
                <td>
                  <span class="font-mono text-xs text-text-secondary bg-surface px-2 py-0.5 rounded-md">
                    {sale.invoice_number}
                  </span>
                </td>
                <td class="text-text-muted text-xs">
                  {new Date(sale.created_at).toLocaleString('id-ID')}
                </td>
                <td class="text-text-secondary">{sale.items?.length || 0} items</td>
                <td>
                  <span class="capitalize text-text-secondary text-sm">{sale.payment_method || '—'}</span>
                </td>
                <td>
                  <Badge variant={statusVariant(sale.status)}>
                    {sale.status || 'completed'}
                  </Badge>
                </td>
                <td class="text-right font-semibold text-text-primary">
                  Rp {(sale.total_amount || 0).toLocaleString('id-ID')}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>