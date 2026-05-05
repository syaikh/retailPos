<script>
  import { onMount } from 'svelte';
  import { goto } from '$lib/router';
  import { auth } from '$lib/stores/auth';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import StatCard from '$lib/components/ui/StatCard.svelte';
  import {
    ShoppingCart, Package, BarChart3, Users,
    TrendingUp, AlertTriangle, Receipt,
    ArrowRight,
  } from 'lucide-svelte';
  import RpIcon from '$lib/components/ui/RpIcon.svelte';

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
      toast.error('Failed to load stats');
    } finally {
      loading = false;
    }
  }

  onMount(fetchStats);

  // Format date and time in Indonesian format: dd mmm yyyy hh:mm:ss
  const formatIndonesianDateTime = (date) => {
    const day = date.getDate().toString().padStart(2, '0');
    const month = date.toLocaleString('id-ID', { month: 'short' });
    const year = date.getFullYear();
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    const seconds = date.getSeconds().toString().padStart(2, '0');

    return `${day} ${month} ${year} ${hours}:${minutes}:${seconds}`;
  };

  // Quick-access modules
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
  <!-- KPI Stats -->
  <div class="card p-6 rounded-2xl border-border">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
    <StatCard
      label="Today's Revenue"
      value={stats.todays_revenue?.toLocaleString('id-ID') || 0}
      sub={stats.todays_revenue > 0 ? "Invoiced today" : "No sales yet today"}
      icon={RpIcon}
      iconBg="bg-primary-subtle"
      iconColor="text-primary-light"
      {loading}
    />
    <StatCard
      label="Transactions"
      value={stats.todays_sales?.toLocaleString('id-ID') || 0}
      sub="Completed today"
      icon={RpIcon}
      iconBg="bg-success-subtle"
      iconColor="text-success-light"
      {loading}
    />
    <StatCard
      label="Total Products"
      value={stats.total_products?.toLocaleString('id-ID') || 0}
      sub="Units in catalog"
      icon={TrendingUp}
      iconBg="bg-info-subtle"
      iconColor="text-info-light"
      {loading}
    />
    <StatCard
      label="Low Stock Alerts"
      value={stats.low_stock_count?.toLocaleString('id-ID') || 0}
      sub={stats.low_stock_count > 0 ? "Action required" : "All stock healthy"}
      icon={AlertTriangle}
      iconBg="bg-warning-subtle"
      iconColor="text-warning-light"
      {loading}
    />
    </div>
  </div>

  <!-- Module cards -->
  <div>
    <h2 class="text-sm font-semibold text-text-muted uppercase tracking-widest mb-4">Quick Access</h2>
    <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
      {#each modules as mod}
        <button
          onclick={() => goto(mod.href)}
          class="card-hover p-5 text-left group bg-gradient-to-br {mod.gradient} border-border cursor-pointer"
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

  <!-- System status -->
  <div class="card p-4 flex items-center gap-4">
    <div class="flex items-center gap-2">
      <span class="w-2.5 h-2.5 bg-success rounded-full animate-pulse-dot"></span>
      <span class="text-xs font-semibold uppercase tracking-wide text-success-light">System Operational</span>
    </div>
    <div class="w-px h-4 bg-border"></div>
    <p class="text-xs text-text-muted">Frontend loaded • Backend connection active</p>
    <div class="ml-auto text-xs text-text-muted">
      {formatIndonesianDateTime(new Date())}
    </div>
  </div>
</div>
