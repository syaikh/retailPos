<script>
  function openPOS() {
    window.location.href = '/pos';
  }
  
  function openInventory() {
    window.location.href = '/inventory';
  }
  
  function openReports() {
    window.location.href = '/reports';
  }
  
  function openAdmin() {
    window.location.href = '/admin';
  }
</script>

<div class="p-6 bg-bg min-h-screen">
  <header class="mb-8">
    <h1 class="text-3xl font-bold bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent mb-2">
      Retail POS System
    </h1>
    <p class="text-slate-400">Ringkasan real-time</p>
  </header>

  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
    <div 
      class="bg-slate-800/50 backdrop-blur-sm border border-slate-700 rounded-xl p-6 cursor-pointer hover:translate-y-[-4px] hover:shadow-xl hover:shadow-blue-500/10 transition-all"
      onclick={openPOS}
      role="button"
      tabindex="0"
      onkeydown={(e) => e.key === 'Enter' && openPOS()}
    >
      <span class="text-4xl mb-4 block">🛒</span>
      <h3 class="text-lg font-bold text-white mb-2">Point of Sale</h3>
      <p class="text-slate-400 text-sm">Proses transaksi dan kelola penjualan</p>
      <button class="btn btn-primary w-full mt-4">Open POS</button>
    </div>

    <div 
      class="bg-slate-800/50 backdrop-blur-sm border border-slate-700 rounded-xl p-6 cursor-pointer hover:translate-y-[-4px] hover:shadow-xl hover:shadow-blue-500/10 transition-all"
      onclick={openInventory}
      role="button"
      tabindex="0"
      onkeydown={(e) => e.key === 'Enter' && openInventory()}
    >
      <span class="text-4xl mb-4 block">📦</span>
      <h3 class="text-lg font-bold text-white mb-2">Inventory</h3>
      <p class="text-slate-400 text-sm">Kelola produk, stok, dan kategori</p>
      <button class="btn btn-primary w-full mt-4">View Inventory</button>
    </div>

    <div 
      class="bg-slate-800/50 backdrop-blur-sm border border-slate-700 rounded-xl p-6 cursor-pointer hover:translate-y-[-4px] hover:shadow-xl hover:shadow-blue-500/10 transition-all"
      onclick={openReports}
      role="button"
      tabindex="0"
      onkeydown={(e) => e.key === 'Enter' && openReports()}
    >
      <span class="text-4xl mb-4 block">📊</span>
      <h3 class="text-lg font-bold text-white mb-2">Reports</h3>
      <p class="text-slate-400 text-sm">Lihat analitik penjualan dan laporan</p>
      <button class="btn btn-primary w-full mt-4">View Reports</button>
    </div>

    <div 
      class="bg-slate-800/50 backdrop-blur-sm border border-slate-700 rounded-xl p-6 cursor-pointer hover:translate-y-[-4px] hover:shadow-xl hover:shadow-blue-500/10 transition-all"
      onclick={openAdmin}
      role="button"
      tabindex="0"
      onkeydown={(e) => e.key === 'Enter' && openAdmin()}
    >
      <span class="text-4xl mb-4 block">⚙️</span>
      <h3 class="text-lg font-bold text-white mb-2">Administration</h3>
      <p class="text-slate-400 text-sm">Kelola pengguna, role, dan pengaturan</p>
      <button class="btn btn-primary w-full mt-4">Open Admin</button>
    </div>
  </div>

  <div class="mt-8 p-4 bg-emerald-500/10 border border-emerald-500/20 rounded-lg">
    <div class="flex items-center gap-2 text-emerald-400 mb-1">
      <span class="w-2 h-2 bg-emerald-400 rounded-full"></span>
      <span class="text-xs font-semibold uppercase">System Status</span>
    </div>
    <h3 class="text-lg font-semibold text-white">Operational</h3>
    <p class="text-slate-400 text-sm mt-1">Frontend loaded successfully • Backend connection active</p>
  </div>
</div>
