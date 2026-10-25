<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/router";
  import { apiFetch } from "$shared/api/http-client";
  import { StatCard, RpIcon } from "$shared/ui";
  import {
    ShoppingCart,
    Package,
    BarChart3,
    Users,
    AlertTriangle,
    ArrowRight,
    HandCoins,
    Tags,
  } from "lucide-svelte";
  import { useWebSocket } from "$shared/api/websocket";
  import { labels } from "$shared/i18n";
  import { useAuthStore } from "$shared/stores/auth.svelte";
  import { useRBAC } from "$shared/composables/useRBAC.svelte";
  import { Permissions } from "$shared/constants/permissions";
  import { PendingSettlementsModal } from "$modules/consignment";
  import type { Component } from "svelte";

  const RpIconComp = RpIcon as unknown as Component;
  const ShoppingCartComp = ShoppingCart as unknown as Component;
  const PackageComp = Package as unknown as Component;
  const BarChart3Comp = BarChart3 as unknown as Component;
  const UsersComp = Users as unknown as Component;
  const AlertTriangleComp = AlertTriangle as unknown as Component;
  const HandCoinsComp = HandCoins as unknown as Component;
  const TagsComp = Tags as unknown as Component;

  const auth = useAuthStore();
  // Card visibility uses the shared RBAC composable — the same source the
  // route guard consults — instead of a second, local permission copy that had
  // drifted from `routePermissions` (cards could render, then bounce with
  // "no permission" on click).
  const rbac = useRBAC();

  let todaysRevenue = $state(0);
  let todaysSales = $state(0);
  let outOfStockCount = $state(0);
  let categoriesCount = $state(0);
  let loading = $state(true);
  let wsConnected = $state(false);
  let showPendingSettlements = $state(false);

  const ws = useWebSocket();
  const revSubText = $derived(
    todaysRevenue > 0 ? labels.invoicedToday : labels.noSalesYetToday,
  );

  async function fetchLiveStats() {
    try {
      const res = await apiFetch("/api/dashboard/live");
      if (res.ok) {
        const data = await res.json();
        if (data.data) {
          todaysRevenue = data.data.todays_revenue || 0;
          todaysSales = data.data.todays_sales || 0;
          outOfStockCount = data.data.out_of_stock_count || 0;
          categoriesCount = data.data.categories_count || 0;
        }
      }
    } catch (_err) {
      // ignore
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    fetchLiveStats();
    const handlers = [
      ws.status.subscribe((status) => {
        wsConnected = status === "connected";
      }),
      ws.on("sale_created", (data: { total?: number }) => {
        if (data && data.total != null) {
          todaysRevenue += data.total;
          todaysSales += 1;
        }
      }),
    ];
    return () => {
      handlers.forEach((fn) => fn());
    };
  });

  const modules = $derived(
    [
      {
        label: labels.pointOfSale,
        desc: labels.posDesc,
        href: "/pos",
        icon: ShoppingCartComp,
        iconBg: "bg-primary-subtle",
        iconColor: "text-primary-light",
        gradient: "from-primary/10 to-accent/5",
        required: [Permissions.sale.create],
      },
      {
        label: labels.inventory,
        desc: labels.inventoryDesc,
        href: "/inventory/products",
        icon: PackageComp,
        iconBg: "bg-success-subtle",
        iconColor: "text-success-light",
        gradient: "from-success/10 to-emerald-600/5",
        required: [Permissions.product.view],
      },
      {
        label: labels.reports,
        desc: labels.reportsDesc,
        href: "/reports",
        icon: BarChart3Comp,
        iconBg: "bg-info-subtle",
        iconColor: "text-info-light",
        gradient: "from-info/10 to-sky-600/5",
        required: [Permissions.report.view],
      },
      {
        label: labels.administration,
        desc: labels.administrationDesc,
        href: "/admin",
        icon: UsersComp,
        iconBg: "bg-warning-subtle",
        iconColor: "text-warning-light",
        gradient: "from-warning/10 to-amber-600/5",
        // /admin requires user.view (routePermissions) — matching it here keeps
        // the card from rendering for users who would be bounced on click.
        required: [Permissions.user.view],
      },
    ].filter((m) => rbac.canAny(m.required)),
  );

  const financeQuickAccess = $derived(
    // /consignment requires consignment.view — gate the card on that same
    // code rather than consignment.pay, which could show the card to users the
    // route then rejects.
    rbac.can(Permissions.consignment.view)
      ? {
          label: "Pending Settlements",
          desc: "Settle supplier payments",
          icon: HandCoinsComp,
          iconBg: "bg-warning-subtle",
          iconColor: "text-warning-light",
          gradient: "from-warning/10 to-amber-600/5",
        }
      : null,
  );

  const visibleStatCards = $derived.by(() => {
    const user = auth.user;
    const role = typeof user?.role === "string" ? user.role : user?.role?.name;

    const showCategories = role === "superadmin" || role === "manager";
    const showOutOfStock =
      role === "superadmin" || role === "manager" || role === "supervisor";

    return {
      showCategories,
      showOutOfStock,
    };
  });
</script>

<div class="space-y-8">
  <div class="card p-6 rounded-2xl border-border">
    <div class="flex items-center justify-between mb-6">
      <h2
        class="text-sm font-semibold text-text-muted uppercase tracking-widest"
      >
        {labels.liveDashboard}
      </h2>
      <span
        class="inline-flex items-center gap-2 rounded-full border border-border px-3 py-1.5 text-xs font-semibold text-text-muted"
      >
        <span class="relative inline-flex h-2 w-2">
          <span
            class="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 {wsConnected
              ? 'bg-success'
              : 'bg-text-muted'}"
          ></span>
          <span
            class="relative inline-flex rounded-full h-2 w-2 {wsConnected
              ? 'bg-success'
              : 'bg-text-muted'}"
          ></span>
        </span>
        {wsConnected ? labels.live : labels.offline}
      </span>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
      <div class="animate-slide-up" style="animation-delay: 100ms;">
        <StatCard
          label={labels.todayRevenue}
          value={loading ? "—" : todaysRevenue?.toLocaleString("id-ID") || 0}
          sub={revSubText}
          icon={RpIconComp}
          iconBg="bg-primary-subtle"
          iconColor="text-primary-light"
          {loading}
        />
      </div>
      <div class="animate-slide-up" style="animation-delay: 200ms;">
        <StatCard
          label={labels.transactionsCard}
          value={loading ? "—" : todaysSales?.toLocaleString("id-ID") || 0}
          sub={todaysSales > 0
            ? labels.completedToday
            : labels.noTransactionsToday}
          icon={ShoppingCartComp}
          iconBg="bg-success-subtle"
          iconColor="text-success-light"
          {loading}
        />
      </div>
      {#if visibleStatCards.showCategories}
        <div class="animate-slide-up" style="animation-delay: 300ms;">
          <StatCard
            label="Categories"
            value={loading
              ? "—"
              : categoriesCount?.toLocaleString("id-ID") || 0}
            sub="Active product categories"
            icon={TagsComp}
            iconBg="bg-info-subtle"
            iconColor="text-info-light"
            {loading}
          />
        </div>
      {/if}
      {#if visibleStatCards.showOutOfStock}
        <div class="animate-slide-up" style="animation-delay: 350ms;">
          <StatCard
            label={labels.outOfStock}
            value={loading
              ? "—"
              : outOfStockCount?.toLocaleString("id-ID") || 0}
            sub={outOfStockCount > 0
              ? labels.actionRequired
              : labels.allItemsInStock}
            icon={AlertTriangleComp}
            iconBg="bg-warning-subtle"
            iconColor="text-warning-light"
            {loading}
          />
        </div>
      {/if}
    </div>
  </div>

  <div>
    <h2
      class="text-sm font-semibold text-text-muted uppercase tracking-widest mb-4"
    >
      {labels.quickAccess}
    </h2>
    <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
      {#if financeQuickAccess}
        <button
          type="button"
          onclick={() => (showPendingSettlements = true)}
          class="card-glass hover:-translate-y-1 transition-all p-5 text-left group bg-linear-to-br {financeQuickAccess.gradient} border-border cursor-pointer animate-slide-up"
          style="animation-delay: 0ms"
        >
          <div class="flex items-start justify-between mb-4">
            <div
              class="w-11 h-11 rounded-xl {financeQuickAccess.iconBg} flex items-center justify-center"
            >
              <financeQuickAccess.icon
                size={22}
                class={financeQuickAccess.iconColor}
              />
            </div>
            <ArrowRight
              size={16}
              class="text-text-muted group-hover:text-text-primary group-hover:translate-x-0.5 transition-all"
            />
          </div>
          <h3 class="font-semibold text-text-primary mb-1">
            {financeQuickAccess.label}
          </h3>
          <p class="text-xs text-text-muted leading-snug">
            {financeQuickAccess.desc}
          </p>
        </button>
      {/if}
      {#each modules as mod, index (index)}
        <button
          type="button"
          onclick={() => goto(mod.href)}
          class="card-glass hover:-translate-y-1 transition-all p-5 text-left group bg-linear-to-br {mod.gradient} border-border cursor-pointer animate-slide-up"
          style="animation-delay: {index * 100 + 100}ms"
        >
          <div class="flex items-start justify-between mb-4">
            <div
              class="w-11 h-11 rounded-xl {mod.iconBg} flex items-center justify-center"
            >
              <mod.icon size={22} class={mod.iconColor} />
            </div>
            <ArrowRight
              size={16}
              class="text-text-muted group-hover:text-text-primary group-hover:translate-x-0.5 transition-all"
            />
          </div>
          <h3 class="font-semibold text-text-primary mb-1">{mod.label}</h3>
          <p class="text-xs text-text-muted leading-snug">{mod.desc}</p>
        </button>
      {/each}
    </div>
  </div>
</div>

<PendingSettlementsModal bind:show={showPendingSettlements} />
