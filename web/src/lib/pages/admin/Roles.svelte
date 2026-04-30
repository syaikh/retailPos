<script>
  let loading = $state(true);
  let roles = $state([]);
</script>

<div class="p-6 bg-bg min-h-screen">
  <h2 class="text-2xl font-bold text-white mb-4">Kelola Role</h2>
  <p class="text-slate-400 mb-6">Manajemen role dan permissions</p>

  {#if loading}
    <div class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
    </div>
  {:else}
    <div class="card p-6">
      <p class="text-slate-400">Roles management content</p>
    </div>
  {/if}
</div>