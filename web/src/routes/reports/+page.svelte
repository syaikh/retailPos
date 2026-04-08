<script>
  import { 
    BarChart3, 
    Download, 
    Calendar,
    Filter,
    ArrowUpRight
  } from 'lucide-svelte';

  const reportData = [
    { date: '2024-04-01', sales: 4500000, items: 120, avg: 37500 },
    { date: '2024-04-02', sales: 3200000, items: 85, avg: 37647 },
    { date: '2024-04-03', sales: 5100000, items: 140, avg: 36428 },
    { date: '2024-04-04', sales: 2800000, items: 70, avg: 40000 },
    { date: '2024-04-05', sales: 6200000, items: 165, avg: 37575 },
  ];
</script>

<div class="reports">
  <div class="header">
    <div class="title">
      <BarChart3 size={32} color="var(--primary)" />
      <h1>Laporan Penjualan</h1>
    </div>
    <button class="export-btn">
      <Download size={18} />
      Ekspor PDF/Excel
    </button>
  </div>

  <div class="filter-bar premium-card glass">
    <div class="filter-item">
      <Calendar size={18} />
      <span>7 Hari Terakhir</span>
    </div>
    <div class="filter-item">
      <Filter size={18} />
      <span>Semua Kategori</span>
    </div>
  </div>

  <div class="chart-placeholder premium-card glass">
    <div class="placeholder-content">
      <BarChart3 size={64} opacity={0.2} />
      <p>Visualisasi Grafik Penjualan</p>
      <span>Pilih rentang tanggal untuk melihat statistik detail</span>
    </div>
  </div>

  <div class="table-container premium-card">
    <table>
      <thead>
        <tr>
          <th>Tanggal</th>
          <th>Total Penjualan</th>
          <th>Item Terjual</th>
          <th>Rata-rata Transaksi</th>
          <th>Aksi</th>
        </tr>
      </thead>
      <tbody>
        {#each reportData as row}
          <tr>
            <td>{row.date}</td>
            <td><strong>Rp {row.sales.toLocaleString()}</strong></td>
            <td>{row.items} unit</td>
            <td>Rp {row.avg.toLocaleString()}</td>
            <td>
              <button class="detail-link">
                Detail <ArrowUpRight size={14} />
              </button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>

<style>
  .reports {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .title {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .export-btn {
    background: var(--bg-surface);
    border: 1px solid var(--border);
    color: var(--text-primary);
    padding: 10px 20px;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .filter-bar {
    display: flex;
    gap: 24px;
    padding: 16px 24px;
  }

  .filter-item {
    display: flex;
    align-items: center;
    gap: 10px;
    color: var(--text-secondary);
    font-size: 0.875rem;
    cursor: pointer;
  }

  .filter-item:hover {
    color: var(--primary);
  }

  .chart-placeholder {
    height: 300px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 2px dashed var(--border);
  }

  .placeholder-content {
    text-align: center;
    color: var(--text-secondary);
  }

  .placeholder-content p {
    font-size: 1.125rem;
    font-weight: 600;
    margin: 16px 0 4px;
  }

  .placeholder-content span {
    font-size: 0.875rem;
  }

  .detail-link {
    background: transparent;
    color: var(--primary);
    font-size: 0.875rem;
    display: flex;
    align-items: center;
    gap: 4px;
  }
</style>
