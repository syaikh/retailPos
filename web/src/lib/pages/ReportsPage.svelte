<script>
  let loading = $state(true);
  let salesData = $state([]);
</script>

<div class="p-6 bg-bg min-h-screen">
  <h2 class="text-2xl font-bold text-white mb-4">Laporan Penjualan</h2>
  <p class="text-slate-400 mb-6">Analitik dan laporan bisnis</p>

  {#if loading}
    <div class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
    </div>
  {:else}
    <div class="card p-6">
      <p class="text-slate-400">Reports content</p>
    </div>
  {/if}
</div>