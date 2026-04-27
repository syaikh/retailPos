<script>
  import { onMount } from "svelte";
  import client from "$lib/api/client";
  import { ShoppingCart, Package, TrendingUp } from "lucide-svelte";
  import { products } from "$lib/stores";
  import Pagination from "$lib/components/Pagination.svelte";

  let stats = $state({
    todaySales: 0,
    todaySalesTrend: 0,
    monthSales: 0,
    monthSalesTrend: 0,
    transactionCount: 0,
    lowStockCount: 0,
    /** @type {any[]} */
    recentActivities: [],
  });

  /** @type {any[]} */
  let lowStockProducts = $state([]);
  let lowStockTotal = $state(0);
  let lowStockLimit = $state(10);
  let lowStockOffset = $state(0);
  let lowStockLoading = $state(false);
  let loading = $state(true);

  onMount(async () => {
    loading = true;
    try {
      const statsResp = await client.get("/stats");
      const data = statsResp.data;

      stats.todaySales = data.today_sales;
      stats.todaySalesTrend = data.today_sales_trend;
      stats.monthSales = data.month_sales;
      stats.monthSalesTrend = data.month_sales_trend;
      stats.transactionCount = data.today_transactions;
      stats.lowStockCount = data.low_stock_count;
      stats.recentActivities = data.recent_activities || [];

      fetchLowStock();

      const productsResp = await client.get("/products");
      products.set(
        Array.isArray(productsResp.data.data) ? productsResp.data.data : [],
      );
    } catch (e) {
      console.error("Failed to fetch dashboard data:", e);
    } finally {
      loading = false;
    }
  });

  async function fetchLowStock() {
    lowStockLoading = true;
    try {
      const resp = await client.get(
        `/products?limit=${lowStockLimit}&offset=${lowStockOffset}&sortBy=stock&sortDir=asc&maxStock=10`,
      );
      lowStockProducts = resp.data.data || [];
      lowStockTotal = resp.data.total || 0;
    } catch (e) {
      console.error("Failed to fetch low stock products:", e);
    } finally {
      lowStockLoading = false;
    }
  }

  function handleLowStockPageChange(
    /** @type {number} */ newOffset,
    /** @type {number | undefined} */ newLimit,
  ) {
    if (newLimit !== undefined) lowStockLimit = newLimit;
    lowStockOffset = newOffset;
    fetchLowStock();
  }

  function formatTimeAgo(/** @type {string} */ dateString) {
    const date = new Date(dateString);
    const now = new Date();
    const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

    if (seconds < 60) return "Baru saja";
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes} menit yang lalu`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours} jam yang lalu`;
    return date.toLocaleDateString();
  }

  function formatMessageWithLinks(/** @type {string} */ message) {
    return message.replace(
      /(#TRX-[A-Z0-9]+)/g,
      '<a href="/reports" class="trx-link">$1</a>',
    );
  }
</script>

<div class="dashboard">
  <div class="header">
    <h1>Ringkasan Bisnis</h1>
    <p>Selamat datang kembali di panel kontrol Anda</p>
  </div>

  <div class="stats-grid">
    <div class="stat-card premium-card glass">
      <div
        class="stat-icon"
        style="background: rgba(16, 185, 129, 0.1); color: var(--success)"
      >
        <span class="icon-rp">Rp</span>
      </div>
      <div class="stat-info">
        <span class="label">Penjualan Hari Ini</span>
        <span class="value">{stats.todaySales.toLocaleString()}</span>
        <span
          class="trend"
          class:positive={stats.todaySalesTrend > 0}
          class:negative={stats.todaySalesTrend < 0}
        >
          <TrendingUp
            size={14}
            style="transform: {stats.todaySalesTrend < 0
              ? 'rotate(180deg)'
              : 'none'}"
          />
          {Math.abs(stats.todaySalesTrend).toFixed(1)}%
        </span>
      </div>
    </div>

    <div class="stat-card premium-card glass">
      <div
        class="stat-icon"
        style="background: rgba(99, 102, 241, 0.1); color: var(--primary)"
      >
        <ShoppingCart size={24} />
      </div>
      <div class="stat-info">
        <span class="label">Transaksi</span>
        <span class="value">{stats.transactionCount}</span>
        <span class="trend">Hari ini</span>
      </div>
    </div>

    <div class="stat-card premium-card glass">
      <div
        class="stat-icon"
        style="background: rgba(239, 68, 68, 0.1); color: var(--danger)"
      >
        <Package size={24} />
      </div>
      <div class="stat-info">
        <span class="label">Stok Menipis</span>
        <span class="value">{stats.lowStockCount} Items</span>
        <span class="trend negative">Perlu Restock</span>
      </div>
    </div>

    <div class="stat-card premium-card glass">
      <div
        class="stat-icon"
        style="background: rgba(34, 211, 238, 0.1); color: var(--accent)"
      >
        <TrendingUp size={24} />
      </div>
      <div class="stat-info">
        <span class="label">Penjualan Bulan Ini</span>
        <span class="value">{(stats.monthSales / 1000000).toFixed(2)} JT</span>
        <span
          class="trend"
          class:positive={stats.monthSalesTrend > 0}
          class:negative={stats.monthSalesTrend < 0}
        >
          {stats.monthSalesTrend > 0 ? "+" : ""}{stats.monthSalesTrend.toFixed(
            1,
          )}%
        </span>
      </div>
    </div>
  </div>

  <div class="recent-grid">
    <div class="premium-card">
      <h3>Produk Stok Terendah</h3>
      <div class="table-container">
        <table>
          <thead>
            <tr>
              <th>SKU</th>
              <th>Barcode</th>
              <th>Nama</th>
              <th>Stok</th>
            </tr>
          </thead>
          <tbody>
            {#each lowStockProducts as p}
              <tr>
                <td><code>{p.sku}</code></td>
                <td>
                  {#if p.barcode}
                    <code>{p.barcode}</code>
                  {:else}
                    <span class="text-dim">-</span>
                  {/if}
                </td>
                <td>{p.name}</td>
                <td><span class="low-stock-text">{p.stock}</span></td>
              </tr>
            {/each}
          </tbody>
        </table>
        {#if !lowStockLoading && lowStockProducts.length === 0}
          <div class="empty-state">
            <Package size={48} opacity={0.2} />
            <p>Semua stok produk mencukupi!</p>
          </div>
        {/if}
      </div>
      <Pagination
        total={lowStockTotal}
        limit={lowStockLimit}
        offset={lowStockOffset}
        onPageChange={handleLowStockPageChange}
      />
    </div>

    <div class="premium-card">
      <h3>Aktivitas Kasir Terbaru</h3>
      <div class="activity-list">
        {#each stats.recentActivities as activity}
          <div class="activity-item">
            {#if activity.type === "sale"}
              <ShoppingCart size={18} class="text-secondary" />
            {:else}
              <Package size={18} class="text-secondary" />
            {/if}
            <div class="details">
              <p>
                <strong>{activity.user}</strong>
                {@html formatMessageWithLinks(activity.message)}
              </p>
              <span>{formatTimeAgo(activity.created_at)}</span>
            </div>
          </div>
        {/each}
        {#if !loading && stats.recentActivities.length === 0}
          <div class="empty-state">
            <p>Tidak ada aktivitas terbaru</p>
          </div>
        {/if}
      </div>
      <div class="card-footer">
        <a href="/reports" class="see-all-link">Lihat laporan selengkapnya →</a>
      </div>
    </div>
  </div>
</div>

<style>
  .dashboard {
    display: flex;
    flex-direction: column;
    gap: 32px;
  }

  .header h1 {
    font-size: 2rem;
    font-weight: 800;
  }

  .header p {
    color: var(--text-secondary);
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: 24px;
  }

  .stat-card {
    display: flex;
    align-items: center;
    gap: 20px;
    padding: 24px;
  }

  .stat-icon {
    width: 56px;
    height: 56px;
    border-radius: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .icon-rp {
    font-size: 1rem;
    font-weight: 800;
    letter-spacing: -0.03em;
    line-height: 1;
  }

  .stat-info {
    display: flex;
    flex-direction: column;
  }

  .label {
    font-size: 0.875rem;
    color: var(--text-secondary);
    font-weight: 500;
  }

  .value {
    font-size: 1.25rem;
    font-weight: 800;
    margin: 4px 0;
    white-space: nowrap;
  }

  .trend {
    font-size: 0.75rem;
    display: flex;
    align-items: center;
    gap: 4px;
    color: var(--text-secondary);
  }

  .trend.positive {
    color: var(--success);
  }
  .trend.negative {
    color: var(--danger);
  }

  .recent-grid {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: 24px;
  }

  h3 {
    margin-bottom: 20px;
    font-size: 1.125rem;
  }

  code {
    background: var(--bg-main);
    padding: 2px 6px;
    border-radius: 4px;
    color: var(--accent);
  }

  .low-stock-text {
    color: var(--danger);
    font-weight: 700;
  }

  .activity-list {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .activity-item {
    display: flex;
    gap: 16px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--border);
  }

  .activity-item:last-child {
    border-bottom: none;
  }

  .details p {
    font-size: 0.875rem;
    margin-bottom: 2px;
  }

  .details span {
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  :global(.trx-link) {
    color: var(--primary);
    text-decoration: none;
    font-weight: 600;
    border-bottom: 1px dashed var(--primary);
    transition:
      color 0.2s,
      border-color 0.2s;
  }

  :global(.trx-link:hover) {
    color: var(--accent);
    border-color: var(--accent);
  }

  .card-footer {
    margin-top: 16px;
    padding-top: 14px;
    border-top: 1px solid var(--border);
    text-align: right;
  }

  .see-all-link {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--primary);
    text-decoration: none;
    letter-spacing: 0.01em;
    transition:
      color 0.2s,
      gap 0.2s;
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  .see-all-link:hover {
    color: var(--accent);
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 40px;
    gap: 16px;
    color: var(--text-secondary);
  }

  .empty-state p {
    font-weight: 500;
  }
</style>

