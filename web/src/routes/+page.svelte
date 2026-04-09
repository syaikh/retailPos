<script>
  import { onMount } from 'svelte';
  import api from '$lib/api.js';
  import { 
    DollarSign, 
    ShoppingCart, 
    Package, 
    TrendingUp,
    Users
  } from 'lucide-svelte';
  import { products } from '$lib/stores.js';

  let stats = $state({
    todaySales: 0,
    monthSales: 0,
    transactionCount: 0,
    lowStockItems: 0
  });

  onMount(async () => {
    // In a real app, these would come from a /stats or /dashboard endpoint
    // For now, we'll fetch products to count low stock
    try {
      const resp = await api.get('/products');
      const data = Array.isArray(resp.data) ? resp.data : [];
      products.set(data);
      stats.lowStockItems = data.filter(p => p.stock < 10).length;
    } catch (e) {
      console.error('Failed to fetch products:', e);
      products.set([]);
    }
    
    // Placeholder stats
    stats.todaySales = 2450000;
    stats.transactionCount = 42;
    stats.monthSales = 125000000;
  });
</script>

<div class="dashboard">
  <div class="header">
    <h1>Ringkasan Bisnis</h1>
    <p>Selamat datang kembali di panel kontrol Anda</p>
  </div>

  <div class="stats-grid">
    <div class="stat-card premium-card glass">
      <div class="stat-icon" style="background: rgba(16, 185, 129, 0.1); color: var(--success)">
        <DollarSign size={24} />
      </div>
      <div class="stat-info">
        <span class="label">Penjualan Hari Ini</span>
        <span class="value">Rp {stats.todaySales.toLocaleString()}</span>
        <span class="trend positive"><TrendingUp size={14} /> +12%</span>
      </div>
    </div>

    <div class="stat-card premium-card glass">
      <div class="stat-icon" style="background: rgba(99, 102, 241, 0.1); color: var(--primary)">
        <ShoppingCart size={24} />
      </div>
      <div class="stat-info">
        <span class="label">Transaksi</span>
        <span class="value">{stats.transactionCount}</span>
        <span class="trend">Hari ini</span>
      </div>
    </div>

    <div class="stat-card premium-card glass">
      <div class="stat-icon" style="background: rgba(239, 68, 68, 0.1); color: var(--danger)">
        <Package size={24} />
      </div>
      <div class="stat-info">
        <span class="label">Stok Menipis</span>
        <span class="value">{stats.lowStockItems} Items</span>
        <span class="trend negative">Perlu Restock</span>
      </div>
    </div>

    <div class="stat-card premium-card glass">
      <div class="stat-icon" style="background: rgba(34, 211, 238, 0.1); color: var(--accent)">
        <TrendingUp size={24} />
      </div>
      <div class="stat-info">
        <span class="label">Penjualan Bulan Ini</span>
        <span class="value">Rp {(stats.monthSales/1000000).toFixed(1)}M</span>
        <span class="trend positive">+5.4%</span>
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
              <th>Nama</th>
              <th>Stok</th>
            </tr>
          </thead>
          <tbody>
            {#each $products && $products.filter(p => p.stock < 10).slice(0, 5) || [] as p}
              <tr>
                <td><code>{p.sku}</code></td>
                <td>{p.name}</td>
                <td><span class="low-stock-text">{p.stock}</span></td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>

    <div class="premium-card">
      <h3>Aktivitas Kasir</h3>
      <div class="activity-list">
        <div class="activity-item">
          <Users size={18} class="text-secondary" />
          <div class="details">
            <p><strong>Admin</strong> memproses transaksi #TRX-9923</p>
            <span>5 menit yang lalu</span>
          </div>
        </div>
        <div class="activity-item">
          <Package size={18} class="text-secondary" />
          <div class="details">
            <p><strong>Admin</strong> menambah stok "Aqua 600ml"</p>
            <span>12 menit yang lalu</span>
          </div>
        </div>
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
    font-size: 1.5rem;
    font-weight: 800;
    margin: 4px 0;
  }

  .trend {
    font-size: 0.75rem;
    display: flex;
    align-items: center;
    gap: 4px;
    color: var(--text-secondary);
  }

  .trend.positive { color: var(--success); }
  .trend.negative { color: var(--danger); }

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
</style>
