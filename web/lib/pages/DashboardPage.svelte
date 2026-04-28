<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { writable } from 'svelte/store';
  import { Card } from '$lib/components/ui';
  import { Badge } from '$lib/components/ui';
  import { useWebSocket } from '$lib/composables/useWebSocket';

  let stats = writable({
    total_sales: 0, total_revenue: 0, total_products: 0, low_stock_count: 0,
    todays_sales: 0, todays_revenue: 0, active_customers: 0
  });
  let loading = true;
  let chartData: any[] = [];
  let recentSales: any[] = [];
  let lowStockItems: any[] = [];
  const ws = useWebSocket();

  async function fetchStats() {
    try {
      const r = await fetch('/api/stats');
      if (r.ok) { const d = await r.json(); stats.set(d.data); }
    } catch(e) { console.error(e); }
  }

  async function fetchChartData() {
    try {
      const r = await fetch('/api/reports/chart');
      if (r.ok) { const d = await r.json(); chartData = d.data || []; }
    } catch(e) { console.error(e); }
  }

  async function fetchRecentSales() {
    try {
      const r = await fetch('/api/sales?limit=5');
      if (r.ok) { const d = await r.json(); recentSales = d.data || []; }
    } catch(e) { console.error(e); }
  }

  async function fetchLowStock() {
    try {
      const r = await fetch('/api/products?maxStock=1');
      if (r.ok) { const d = await r.json(); lowStockItems = d.data || []; }
    } catch(e) { console.error(e); }
  }

  function handleSaleCreated(p: any) {
    stats.update(s => ({ ...s, total_sales: s.total_sales + 1, total_revenue: s.total_revenue + p.total, todays_sales: s.todays_sales + 1, todays_revenue: s.todays_revenue + p.total }));
    fetchRecentSales();
  }

  function handleStockUpdate(p: any) { if (p.low_stock) fetchLowStock(); }

  onMount(async () => {
    await Promise.all([fetchStats(), fetchChartData(), fetchRecentSales(), fetchLowStock()]);
    loading = false;
    ws.wsEvents?.on('sale_created', handleSaleCreated);
    ws.wsEvents?.on('stock_update', handleStockUpdate);
  });

  onDestroy(() => {
    ws.wsEvents?.off('sale_created', handleSaleCreated);
    ws.wsEvents?.off('stock_update', handleStockUpdate);
  });

  function MiniChart({ d, c = '#3b82f6' }: { d: number[], c?: string }) {
    const m = Math.max(...d, 1);
    const pts = d.map((v, i) => i * 10 + ',' + (20 - v / m * 20)).join(' ');
    return `<svg viewBox="0 0 100 25" class="w-full h-6"><polyline points="${pts}" fill="none" stroke="${c}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>`;
  }
</script>

<div class="dashboard p-6 bg-gray-100 min-h-screen">
  <div class="mb-6"><h1 class="text-2xl font-bold text-gray-800">Dashboard</h1><p class="text-gray-600">Ringkasan real-time</p></div>
  {#if loading}
    <div class="flex justify-center py-12"><div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div></div>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
      <Card class="p-5"><div class="w-12 h-12 bg-blue-500 rounded-lg flex items-center justify-center mb-3"><svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6"/></svg></div><p class="text-sm text-gray-500">Total Penjualan</p><p class="text-2xl font-bold">{$stats.total_sales.toLocaleString()}</p><p class="text-xs text-green-600">+{$stats.todays_sales} hari ini</p></Card>
      <Card class="p-5"><div class="w-12 h-12 bg-green-500 rounded-lg flex items-center justify-center mb-3"><svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg></div><p class="text-sm text-gray-500">Pendapatan</p><p class="text-2xl font-bold">Rp{$stats.total_revenue.toLocaleString()}</p><p class="text-xs text-green-600">+Rp{$stats.todays_revenue.toLocaleString()} hari ini</p></Card>
      <Card class="p-5"><div class="w-12 h-12 bg-purple-500 rounded-lg flex items-center justify-center mb-3"><svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg></div><p class="text-sm text-gray-500">Produk</p><p class="text-2xl font-bold">{$stats.total_products}</p><p class="text-xs {$stats.low_stock_count > 0 ? 'text-red-600' : 'text-green-600'}">{$stats.low_stock_count > 0 ? $stats.low_stock_count + ' stok rendah' : 'Stok aman'}</p></Card>
      <Card class="p-5"><div class="w-12 h-12 bg-orange-500 rounded-lg flex items-center justify-center mb-3"><svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0z"/></svg></div><p class="text-sm text-gray-500">Pelanggan Aktif</p><p class="text-2xl font-bold">{$ws.onlineUsers || $stats.active_customers}</p><Badge variant="secondary"><span class="w-2 h-2 bg-green-500 rounded-full mr-1"></span>Online</Badge></Card>
    </div>
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
      <Card class="p-5 lg:col-span-2"><h3 class="text-lg font-semibold mb-4">Tren Penjualan</h3>{#if chartData.length > 0}<div class="h-16 mb-2">{@html MiniChart({d: chartData.map(d => d.sales)})}</div><div class="flex justify-between text-xs text-gray-400">{#each chartData as item}<span>{new Date(item.day).toLocaleDateString('id-ID', {weekday: 'short'})}</span>{/each}</div>{:else}<div class="text-center py-8 text-gray-400">Belum ada data</div>{/if}</Card>
      <Card class="p-5"><h3 class="text-lg font-semibold mb-3">Terbaru</h3><div class="space-y-2 max-h-64 overflow-y-auto">{#each recentSales as s}<div class="flex justify-between text-sm p-2 border rounded"><div><p class="font-medium text-xs">{s.invoice_number}</p><p class="text-gray-400 text-xs">{new Date(s.created_at).toLocaleDateString()}</p></div><span class="font-semibold text-sm">Rp{s.total_amount.toLocaleString()}</span></div>{/each}</div></Card>
      {#if lowStockItems.length > 0}
      <Card class="p-5 lg:col-span-2"><h3 class="text-lg font-semibold text-orange-600 mb-3">Stok Rendah ({lowStockItems.length})</h3><div class="space-y-2"><div class="grid grid-cols-1 md:grid-cols-2 gap-2">{#each lowStockItems as p}<div class="p-3 border border-orange-200 bg-orange-50 rounded"><div class="flex justify-between"><span class="font-medium text-sm">{p.name}</span><span class="text-xs text-gray-400">{p.sku}</span></div><div class="flex justify-between mt-1"><span class="text-xs text-orange-600">Sisa: {p.stock}/{p.stock_min}</span><span class="text-xs px-2 py-0.5 bg-orange-100 text-orange-700 rounded">{Math.round(p.stock/p.stock_min*100)}%</span></div></div>{/each}</div></div></Card>
      {/if}
    </div>
  {/if}
</div>
