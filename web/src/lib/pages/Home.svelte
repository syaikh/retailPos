<script>
  import { onMount } from 'svelte';
  import { goto } from '$lib/router';
  import { apiFetch } from '$lib/api/client';
  import StatCard from '$lib/components/ui/StatCard.svelte';
  import {
    ShoppingCart, Package, BarChart3, Users,
    AlertTriangle,
    ArrowRight,
  } from 'lucide-svelte';
  import RpIcon from '$lib/components/ui/RpIcon.svelte';
  import { useWebSocket } from '$lib/composables/useWebSocket';

  let todaysRevenue = $state(0);
  let todaysSales = $state(0);
  let totalProducts = $state(0);
  let lowStockCount = $state(0);
  let loading = $state(true);
  let wsConnected = $state(false);

  const ws = useWebSocket();
  const revSubText = $derived(todaysRevenue > 0 ? 'Invoiced today' : 'No sales yet today');

  async function fetchLiveStats() {
    try {
      const res = await apiFetch('/api/dashboard/live');
      if (res.ok) {
        const data = await res.json();
        if (data.data) {
          todaysRevenue = data.data.todays_revenue || 0;
          todaysSales = data.data.todays_sales || 0;
          totalProducts = data.data.total_products || 0;
          lowStockCount = data.data.low_stock_count || 0;
        }
      }
    } catch (err) {
      // ignore
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    fetchLiveStats();
    const handlers = [
      ws.status.subscribe((status) => { wsConnected = status === 'connected'; }),
      ws.on('sale_created', (data) => {
        if (data && data.total != null) {
          todaysRevenue += data.total;
          todaysSales += 1;
        }
      }),
    ];
    const intervalId = setInterval(fetchLiveStats, 30000);
    return () => {
      handlers.forEach((fn) => fn());
      clearInterval(intervalId);
    };
  });

  const modules = [
    {
      label: 'Point of Sale',
      desc: 'Process transactions & manage sales',
      href: '/pos',
      icon: ShoppingCart,
      iconBg: 'bg-primary-subtle',
      iconColor: 'text-primary-light',
      gradient: 'from-primary/10 to-accent/5',
    },
    {
      label: 'Inventory',
      desc: 'Manage products, stock & categories',
      href: '/inventory',
      icon: Package,
      iconBg: 'bg-success-subtle',
      iconColor: 'text-success-light',
      gradient: 'from-success/10 to-emerald-600/5',
    },
    {
      label: 'Reports',
      desc: 'Analytics, sales reports & export',
      href: '/reports',
      icon: BarChart3,
      iconBg: 'bg-info-subtle',
      iconColor: 'text-info-light',
      gradient: 'from-info/10 to-sky-600/5',
    },
    {
      label: 'Administration',
      desc: 'Users, roles & system settings',
      href: '/admin',
      icon: Users,
      iconBg: 'bg-warning-subtle',
      iconColor: 'text-warning-light',
      gradient: 'from-warning/10 to-amber-600/5',
    },
  ];
</script>

<div class="space-y-8">
  <div class="card p-6 rounded-2xl border-border">
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-sm font-semibold text-text-muted uppercase tracking-widest">Live Dashboard</h2>
      <span class="inline-flex items-center gap-2 rounded-full border border-border px-3 py-1.5 text-xs font-semibold text-text-muted">
        <span class="relative inline-flex h-2 w-2">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 {wsConnected ? 'bg-success' : 'bg-text-muted'}"></span>
          <span class="relative inline-flex rounded-full h-2 w-2 {wsConnected ? 'bg-success' : 'bg-text-muted'}"></span>
        </span>
        {wsConnected ? 'Live' : 'Offline'}
      </span>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
      <div class="animate-slide-up" style="animation-delay: 100ms;">
        <StatCard
          label="Today's Revenue"
          value={loading ? '—' : (todaysRevenue?.toLocaleString('id-ID') || 0)}
          sub={revSubText}
          icon={RpIcon}
          iconBg="bg-primary-subtle"
          iconColor="text-primary-light"
          {loading}
        />
      </div>
      <div class="animate-slide-up" style="animation-delay: 200ms;">
        <StatCard
          label="Transactions"
          value={loading ? '—' : (todaysSales?.toLocaleString('id-ID') || 0)}
          sub={todaysSales > 0 ? "Completed today" : "No transactions today"}
          icon={ShoppingCart}
          iconBg="bg-success-subtle"
          iconColor="text-success-light"
          {loading}
        />
      </div>
      <div class="animate-slide-up" style="animation-delay: 300ms;">
        <StatCard
          label="Total Products"
          value={loading ? '—' : (totalProducts?.toLocaleString('id-ID') || 0)}
          sub="Units in catalog"
          icon={Package}
          iconBg="bg-info-subtle"
          iconColor="text-info-light"
          {loading}
        />
      </div>
      <div class="animate-slide-up" style="animation-delay: 400ms;">
        <StatCard
          label="Low Stock Alerts"
          value={loading ? '—' : (lowStockCount?.toLocaleString('id-ID') || 0)}
          sub={lowStockCount > 0 ? "Action required" : "All stock healthy"}
          icon={AlertTriangle}
          iconBg="bg-warning-subtle"
          iconColor="text-warning-light"
          {loading}
        />
      </div>
    </div>
  </div>

  <div>
    <h2 class="text-sm font-semibold text-text-muted uppercase tracking-widest mb-4">Quick Access</h2>
    <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
      {#each modules as mod, index}
        <button
          onclick={() => goto(mod.href)}
          class="card-glass hover:-translate-y-1 transition-all p-5 text-left group bg-linear-to-br {mod.gradient} border-border cursor-pointer animate-slide-up"
          style="animation-delay: {index * 100 + 500}ms"
        >
          <div class="flex items-start justify-between mb-4">
            <div class="w-11 h-11 rounded-xl {mod.iconBg} flex items-center justify-center">
              <mod.icon size={22} class={mod.iconColor} />
            </div>
            <ArrowRight size={16} class="text-text-muted group-hover:text-text-primary group-hover:translate-x-0.5 transition-all" />
          </div>
          <h3 class="font-semibold text-text-primary mb-1">{mod.label}</h3>
          <p class="text-xs text-text-muted leading-snug">{mod.desc}</p>
        </button>
      {/each}
    </div>
  </div>
</div>
