<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { auth } from '$lib/stores/auth';
  import StatCard from '$lib/components/ui/StatCard.svelte';
  import { ShoppingCart, Package, BarChart3, Users, AlertTriangle } from 'lucide-svelte';
  import RpIcon from '$lib/components/ui/RpIcon.svelte';

  let username = $derived($auth.user?.username || 'there');
  let stats = $state({
    todays_revenue: 0,
    todays_sales: 0,
    total_products: 0,
    low_stock_count: 0
  });
  let loading = $state(true);

  async function fetchStats() {
    try {
      loading = true;
      const res = await apiFetch('/api/dashboard/stats');
      if (res.ok) {
        const data = await res.json();
        stats = data.data;
      }
    } catch (err) {
      toast.error('Failed to load dashboard stats');
    } finally {
      loading = false;
    }
  }

  onMount(fetchStats);

  const greeting = (() => {
    const h = new Date().getHours();
    if (h < 12) return 'Good morning';
    if (h < 17) return 'Good afternoon';
    return 'Good evening';
  })();

  function goTo(path) {
    window.location.href = path;
  }
</script>

<div class="space-y-8">
  <!-- Greeting -->
  <div>
    <h1 class="text-2xl font-bold text-text-primary">
      {greeting}, <span class="gradient-text capitalize">{username}</span> 👋
    </h1>
    <p class="text-text-muted text-sm mt-1">
      Here's what's happening at your store today.
    </p>
  </div>

  <!-- KPI Stats -->
  <div class="grid grid-cols-2 xl:grid-cols-4 gap-4">
    <StatCard
      label="Today's Revenue"
      value={loading ? '—' : (stats.todays_revenue?.toLocaleString('id-ID') || 0)}
      sub={stats.todays_revenue > 0 ? "Invoiced today" : "No sales yet today"}
      icon={RpIcon}
      iconBg="bg-primary-subtle"
      iconColor="text-primary-light"
      {loading}
    />
    <StatCard
      label="Transactions"
      value={loading ? '—' : (stats.todays_sales || 0)}
      sub="Completed today"
      icon={ShoppingCart}
      iconBg="bg-success-subtle"
      iconColor="text-success-light"
      {loading}
    />
    <StatCard
      label="Total Products"
      value={loading ? '—' : (stats.total_products || 0)}
      sub="Units in catalog"
      icon={Package}
      iconBg="bg-info-subtle"
      iconColor="text-info-light"
      {loading}
    />
    <StatCard
      label="Low Stock Alerts"
      value={loading ? '—' : (stats.low_stock_count || 0)}
      sub={stats.low_stock_count > 0 ? "Action required" : "All stock healthy"}
      icon={AlertTriangle}
      iconBg="bg-warning-subtle"
      iconColor="text-warning-light"
      {loading}
    />
  </div>

  <!-- Quick Actions -->
  <div>
    <h2 class="text-sm font-semibold text-text-muted uppercase tracking-widest mb-4">Quick Access</h2>
    <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
      <button onclick={() => goTo('/pos')} class="card-hover p-5 text-left group bg-gradient-to-br from-primary/10 to-accent/5 border-border">
        <div class="flex items-start justify-between mb-4">
          <div class="w-11 h-11 rounded-xl bg-primary-subtle flex items-center justify-center">
            <ShoppingCart size={22} class="text-primary-light" />
          </div>
        </div>
        <h3 class="font-semibold text-text-primary mb-1">Point of Sale</h3>
        <p class="text-xs text-text-muted leading-snug">Process transactions & manage sales</p>
      </button>

      <button onclick={() => goTo('/inventory')} class="card-hover p-5 text-left group bg-gradient-to-br from-success/10 to-emerald-600/5 border-border">
        <div class="flex items-start justify-between mb-4">
          <div class="w-11 h-11 rounded-xl bg-success-subtle flex items-center justify-center">
            <Package size={22} class="text-success-light" />
          </div>
        </div>
        <h3 class="font-semibold text-text-primary mb-1">Inventory</h3>
        <p class="text-xs text-text-muted leading-snug">Manage products, stock & categories</p>
      </button>

      <button onclick={() => goTo('/reports')} class="card-hover p-5 text-left group bg-gradient-to-br from-info/10 to-sky-600/5 border-border">
        <div class="flex items-start justify-between mb-4">
          <div class="w-11 h-11 rounded-xl bg-info-subtle flex items-center justify-center">
            <BarChart3 size={22} class="text-info-light" />
          </div>
        </div>
        <h3 class="font-semibold text-text-primary mb-1">Reports</h3>
        <p class="text-xs text-text-muted leading-snug">Analytics, sales reports & export</p>
      </button>

      <button onclick={() => goTo('/admin')} class="card-hover p-5 text-left group bg-gradient-to-br from-warning/10 to-amber-600/5 border-border">
        <div class="flex items-start justify-between mb-4">
          <div class="w-11 h-11 rounded-xl bg-warning-subtle flex items-center justify-center">
            <Users size={22} class="text-warning-light" />
          </div>
        </div>
        <h3 class="font-semibold text-text-primary mb-1">Administration</h3>
        <p class="text-xs text-text-muted leading-snug">Users, roles & system settings</p>
      </button>
    </div>
  </div>

  <!-- System Status -->
  <div class="card p-4 flex items-center gap-4">
    <div class="flex items-center gap-2">
      <span class="w-2.5 h-2.5 bg-success rounded-full animate-pulse-dot"></span>
      <span class="text-xs font-semibold uppercase tracking-wide text-success-light">System Operational</span>
    </div>
    <div class="w-px h-4 bg-border"></div>
    <p class="text-xs text-text-muted">Backend connection active</p>
    <div class="ml-auto text-xs text-text-muted">
      {new Date().toLocaleString('en-US', { dateStyle: 'medium', timeStyle: 'medium' })}
    </div>
  </div>
</div>