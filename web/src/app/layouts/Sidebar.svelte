<script lang="ts">
  import { LayoutDashboard, ShoppingCart, Package, BarChart3, Users, Shield, ScrollText, ChevronDown, ChevronLeft, ChevronRight, LogOut, Store, User, Tag, Database, Building2, Ruler, Truck, Percent, Clock, ClipboardList, Warehouse } from 'lucide-svelte';
  import { fly } from 'svelte/transition';
  import { goto, getPath } from '$app/router';
  import { logout, useAuthStore } from '$modules/auth';
  import { useShiftStore } from '$modules/shifts';
  import { Tooltip } from '$shared/ui';
  import { routePermissions } from '$app/config/permissions';
  import { hasPermission } from '$shared/utils/permissions';

  let {
    currentPath = $bindable('/'),
    isMobileMenuOpen = false,
    onclose = () => {},
  }: {
    currentPath?: string;
    isMobileMenuOpen?: boolean;
    onclose?: () => void;
  } = $props();

  let collapsed = $state(false);
  let adminExpanded = $state(false);
  let masterDataExpanded = $state(false);

  const isAdminPath = $derived(currentPath.startsWith('/admin') || currentPath.startsWith('/stores'));
  const isMasterDataPath = $derived(
    currentPath.startsWith('/inventory/products') ||
    currentPath.startsWith('/categories') ||
    currentPath.startsWith('/customers') ||
    currentPath.startsWith('/brands') ||
    currentPath.startsWith('/units-of-measure') ||
    currentPath.startsWith('/pricing-rules') ||
    currentPath.startsWith('/customer-groups') ||
    currentPath.startsWith('/suppliers') ||
    currentPath.startsWith('/storage-locations')
  );

  $effect(() => {
    if (isAdminPath) adminExpanded = true;
  });

  $effect(() => {
    if (isMasterDataPath) masterDataExpanded = true;
  });

  $effect(() => {
    function handleKeydown(e: KeyboardEvent) {
      if (e.key === 'Escape' && isMobileMenuOpen) onclose();
    }
    window.addEventListener('keydown', handleKeydown);
    return () => window.removeEventListener('keydown', handleKeydown);
  });

  const authStore = useAuthStore();
  const shiftStore = useShiftStore();
  let username = $derived(authStore.user?.username || 'User');
  let role = $derived(
    typeof authStore.user?.role === 'object' && authStore.user?.role ? authStore.user.role.name :
    typeof authStore.user?.role === 'string' ? authStore.user.role :
    (authStore.user?.role_id === 1 ? 'superadmin' : authStore.user?.role_id === 2 ? 'admin' : authStore.user?.role_id === 3 ? 'cashier' : authStore.user?.role_id === 4 ? 'manager' : authStore.user?.role_id === 5 ? 'staff' : 'cashier')
  );

  let canLogout = $derived(role !== 'cashier' || !shiftStore.activeShift);

  let userPerms = $derived(authStore.user?.permissions || []);

  function canAccess(href: string): boolean {
    const required = routePermissions[href];
    if (!required || required.length === 0) return true;
    return required.some(p => hasPermission(userPerms, p));
  }

  const navItems: Array<{ label: string; href: string; icon: any; iconText?: string }> = [
    { label: 'Dashboard',     href: '/',                  icon: LayoutDashboard },
    { label: 'Point of Sale', href: '/pos',               icon: ShoppingCart },
    { label: 'Transactions',  href: '/transactions',       icon: undefined, iconText: 'Rp' },
    { label: 'Reports',       href: '/reports',           icon: BarChart3 },
    { label: 'Shifts',        href: '/shifts',             icon: Clock },
    { label: 'Purchase Orders', href: '/purchase-orders',  icon: Truck },
    { label: 'Stock Opname',  href: '/stock-opnames',      icon: ClipboardList },
  ];

  const masterDataSubItems = [
    { label: 'Products',   href: '/inventory/products', icon: Package },
    { label: 'Categories', href: '/categories',          icon: Tag },
    { label: 'Brands',     href: '/brands',        icon: Building2 },
    { label: 'Units',      href: '/units-of-measure', icon: Ruler },
    { label: 'Customers',  href: '/customers',           icon: User },
    { label: 'Pricing Rules', href: '/pricing-rules', icon: Percent },
    { label: 'Customer Groups', href: '/customer-groups', icon: Users },
    { label: 'Suppliers',  href: '/suppliers',           icon: Truck },
    { label: 'Storage Locations', href: '/storage-locations', icon: Warehouse },
  ];

  const managerNavItems: Array<{ label: string; href: string; icon: any; iconText?: string }> = [
    { label: 'Dashboard',     href: '/',                  icon: LayoutDashboard },
    { label: 'Transactions',  href: '/transactions',       icon: undefined, iconText: 'Rp' },
    { label: 'Reports',       href: '/reports',           icon: BarChart3 },
    { label: 'Shifts',        href: '/shifts',             icon: Clock },
    { label: 'Purchase Orders', href: '/purchase-orders',  icon: Truck },
    { label: 'Stock Opname',  href: '/stock-opnames',      icon: ClipboardList },
  ];

  const managerMasterDataSubItems = [
    { label: 'Products',   href: '/inventory/products', icon: Package },
    { label: 'Categories', href: '/categories',          icon: Tag },
    { label: 'Brands',     href: '/brands',        icon: Building2 },
    { label: 'Units',      href: '/units-of-measure', icon: Ruler },
    { label: 'Customers',  href: '/customers',           icon: User },
    { label: 'Pricing Rules', href: '/pricing-rules', icon: Percent },
    { label: 'Customer Groups', href: '/customer-groups', icon: Users },
    { label: 'Suppliers',  href: '/suppliers',           icon: Truck },
    { label: 'Storage Locations', href: '/storage-locations', icon: Warehouse },
  ];

  const cashierNavItems: Array<{ label: string; href: string; icon: any; iconText?: string }> = [
    { label: 'Point of Sale', href: '/pos',               icon: ShoppingCart },
    { label: 'Transactions',  href: '/transactions',       icon: undefined, iconText: 'Rp' },
    { label: 'Shifts',        href: '/shifts',             icon: Clock },
  ];

  const staffNavItems: Array<{ label: string; href: string; icon: any; iconText?: string }> = [
    { label: 'Stock Opname', href: '/stock-opnames', icon: ClipboardList },
  ];

  const staffMasterDataSubItems = [
    { label: 'Products',   href: '/inventory/products', icon: Package },
  ];

  const adminItems = [
    { label: 'Stores',      href: '/stores',       icon: Store },
    { label: 'Users',       href: '/admin/users',       icon: Users },
    { label: 'Roles',       href: '/admin/roles',       icon: Shield },
    { label: 'Audit Logs',  href: '/admin/audit-logs',  icon: ScrollText, requiresSuperadmin: true },
  ];

  let visibleNavItems = $derived(
    (role === 'staff' ? staffNavItems :
    role === 'cashier' ? cashierNavItems :
    (role === 'manager' ? managerNavItems : navItems)
    ).filter(item => canAccess(item.href))
  );

  let visibleMasterDataSubItems = $derived(
    (role === 'staff' ? staffMasterDataSubItems :
    role === 'cashier' ? [] :
    (role === 'manager' ? managerMasterDataSubItems : masterDataSubItems)
    ).filter(item => canAccess(item.href))
  );

  let visibleAdminItems = $derived(
    adminItems.filter(item => canAccess(item.href) && (!item.requiresSuperadmin || role === 'superadmin'))
  );

  let showAdminSection = $derived((role === 'admin' || role === 'superadmin') && visibleAdminItems.length > 0);

  function isActive(href: string) {
    if (href === '/') return currentPath === '/';
    if (href === '/stock-opnames') return currentPath === '/stock-opnames' || currentPath.startsWith('/stock-opnames/');
    return currentPath === href;
  }

  function navigate(href: string) {
    goto(href);
    onclose();
  }

  async function handleLogout() {
    await logout();
  }

  function createRipple(event: MouseEvent, el: HTMLElement) {
    const button = el;
    const circle = document.createElement('span');
    const diameter = Math.max(button.clientWidth, button.clientHeight);
    const radius = diameter / 2;

    const rect = button.getBoundingClientRect();
    const x = event.clientX - rect.left - radius;
    const y = event.clientY - rect.top - radius;

    circle.style.width = circle.style.height = `${diameter}px`;
    circle.style.left = `${x}px`;
    circle.style.top = `${y}px`;
    circle.classList.add('sidebar-ripple');

    const ripple = button.getElementsByClassName('sidebar-ripple')[0];
    if (ripple) ripple.remove();

    button.appendChild(circle);
    setTimeout(() => circle.remove(), 600);
  }
</script>

<aside
  class="sidebar-shell flex flex-col bg-sidebar border-r border-sidebar-border shadow-sidebar shrink-0 transition-all duration-300 ease-spring max-md:fixed max-md:inset-y-0 max-md:left-0 max-md:z-50 max-md:w-[var(--sidebar-width)] {isMobileMenuOpen ? 'max-md:translate-x-0' : 'max-md:-translate-x-full'}"
  style:width={collapsed ? 'var(--sidebar-collapsed-width)' : 'var(--sidebar-width)'}
  aria-label="Sidebar"
>
  <!-- Brand -->
  <div class="flex items-center gap-3 px-4 h-16 border-b border-sidebar-border">
    <div class="w-9 h-9 rounded-xl gradient-bg-primary flex items-center justify-center shrink-0 shadow-glow-primary-sm">
      <Store size={18} class="text-white" />
    </div>
    {#if !collapsed}
      <div class="overflow-hidden">
        <p class="text-sm font-bold text-text-primary leading-tight truncate">RetailPOS</p>
        <p class="text-[10px] text-text-muted truncate">Management System</p>
      </div>
    {/if}
  </div>

  <!-- Nav -->
  <nav class="flex-1 overflow-y-auto overflow-x-hidden py-4 px-2.5 space-y-0.5 no-scrollbar" aria-label="Main navigation">
    {#each visibleNavItems as item}
<button type="button" 
        onclick={(e) => { createRipple(e, e.currentTarget); navigate(item.href); }}
        class={isActive(item.href) ? 'sidebar-item-active w-full text-left relative overflow-hidden px-3 py-2.5' : 'sidebar-item w-full text-left relative overflow-hidden px-3 py-2.5'}
        aria-current={isActive(item.href) ? 'page' : undefined}
        aria-label={collapsed ? item.label : undefined}
        title={collapsed ? item.label : ''}
      >
        {#if item.iconText}
          <span class="text-xs font-bold shrink-0">{item.iconText}</span>
        {:else}
          <item.icon size={18} class="shrink-0" />
        {/if}
        {#if !collapsed}
          <span class="relative z-10">{item.label}</span>
        {/if}
      </button>
    {/each}

    <!-- Master Data group -->
    {#if visibleMasterDataSubItems.length > 0}
      <div class="pt-1" role="group" aria-label="Master Data">
        {#if !collapsed}
          <p class="px-3 pt-3 pb-1 text-[10px] font-semibold uppercase tracking-widest text-text-muted">Master Data</p>
        {/if}
  <button type="button" 
          onclick={(e) => { createRipple(e, e.currentTarget); if (!collapsed) masterDataExpanded = !masterDataExpanded; else navigate('/inventory/products'); }}
          class={isMasterDataPath ? 'sidebar-parent-active w-full text-left relative overflow-hidden px-3 py-2.5' : 'sidebar-item w-full text-left relative overflow-hidden px-3 py-2.5'}
          aria-expanded={masterDataExpanded}
          aria-controls="sidebar-section-master-data"
          aria-label={collapsed ? 'Master Data' : undefined}
          title={collapsed ? 'Master Data' : ''}
        >
          <Database size={18} class="shrink-0" />
          {#if !collapsed}
            <span class="relative z-10 flex-1">Master Data</span>
            <ChevronDown size={14} class="text-text-muted transition-transform duration-200 {masterDataExpanded ? 'rotate-0' : '-rotate-90'}" />
          {/if}
        </button>

        {#if masterDataExpanded && !collapsed}
          <div id="sidebar-section-master-data" transition:fly={{ y: -8, duration: 200, opacity: 0 }} class="pt-0.5">
            {#each visibleMasterDataSubItems as subItem}
        <button type="button" 
                onclick={(e) => { createRipple(e, e.currentTarget); navigate(subItem.href); }}
                class={isActive(subItem.href) ? 'sidebar-item-active w-full text-left relative overflow-hidden py-2.5 pr-3 pl-9' : 'sidebar-item w-full text-left relative overflow-hidden py-2.5 pr-3 pl-9'}
                aria-current={isActive(subItem.href) ? 'page' : undefined}
              >
                <subItem.icon size={16} class="shrink-0" />
                <span class="relative z-10">{subItem.label}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
    {/if}

    <!-- Administration group -->
    {#if showAdminSection}
    <div class="pt-4" role="group" aria-label="Administration">
      {#if !collapsed}
        <p class="px-3 pt-3 pb-1 text-[10px] font-semibold uppercase tracking-widest text-text-muted">Administration</p>
      {/if}
<button type="button" 
        onclick={(e) => { createRipple(e, e.currentTarget); if (!collapsed) adminExpanded = !adminExpanded; else navigate('/admin/users'); }}
        class={isAdminPath ? 'sidebar-parent-active w-full text-left relative overflow-hidden px-3 py-2.5' : 'sidebar-item w-full text-left relative overflow-hidden px-3 py-2.5'}
        aria-expanded={adminExpanded}
        aria-controls="sidebar-section-admin"
        aria-label={collapsed ? 'Administration' : undefined}
        title={collapsed ? 'Administration' : ''}
      >
        <Shield size={18} class="shrink-0" />
        {#if !collapsed}
          <span class="relative z-10 flex-1">Administration</span>
          <ChevronDown size={14} class="text-text-muted transition-transform duration-200 {adminExpanded ? 'rotate-0' : '-rotate-90'}" />
        {/if}
      </button>

      {#if adminExpanded && !collapsed}
        <div id="sidebar-section-admin" transition:fly={{ y: -8, duration: 200, opacity: 0 }} class="pt-0.5">
          {#each visibleAdminItems as item}
      <button type="button" 
              onclick={(e) => { createRipple(e, e.currentTarget); navigate(item.href); }}
              class={isActive(item.href) ? 'sidebar-item-active w-full text-left relative overflow-hidden py-2.5 pr-3 pl-9' : 'sidebar-item w-full text-left relative overflow-hidden py-2.5 pr-3 pl-9'}
              aria-current={isActive(item.href) ? 'page' : undefined}
            >
              <item.icon size={16} class="shrink-0" />
              <span class="relative z-10">{item.label}</span>
            </button>
          {/each}
        </div>
      {/if}
    </div>
    {/if}
  </nav>

  <!-- Bottom: user + collapse toggle -->
  <div class="mt-auto border-t border-sidebar-border px-2.5 py-3 space-y-0.5">
    {#if authStore.isAuthenticated}
    <!-- User row -->
    <div class="flex items-center gap-3 px-3 py-2.5 rounded-xl" title={collapsed ? username : ''}>
      <div class="w-8 h-8 rounded-full gradient-bg-primary flex items-center justify-center shrink-0">
        <User size={14} class="text-white" />
      </div>
      {#if !collapsed}
        <div class="flex-1 min-w-0">
          <p class="text-xs font-semibold text-text-primary truncate">{username}</p>
          <p class="text-[10px] text-text-muted capitalize truncate">{role}</p>
        </div>
        {#if canLogout}
          <button type="button" 
            onclick={handleLogout}
            class="flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-text-muted hover:text-danger hover:bg-danger-subtle transition-all duration-200 group"
            title="Logout"
          >
            <LogOut size={14} class="group-hover:scale-110 transition-transform" />
            <span class="text-xs font-medium">Logout</span>
          </button>
        {:else}
          <Tooltip content="Close shift first" placement="top">
            <span class="flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-text-muted/40 cursor-not-allowed">
              <LogOut size={14} />
              <span class="text-xs font-medium">Logout</span>
            </span>
          </Tooltip>
        {/if}
      {/if}
    </div>

    {#if collapsed}
      {#if canLogout}
        <button type="button" 
          onclick={handleLogout}
          class="sidebar-item w-full justify-center text-text-muted hover:text-danger hover:bg-danger-subtle px-3 py-2.5"
          title="Logout"
          aria-label="Logout"
        >
          <LogOut size={18} />
        </button>
      {:else}
        <Tooltip content="Close shift first" placement="right">
          <span class="sidebar-item w-full justify-center text-text-muted/40 cursor-not-allowed px-3 py-2.5">
            <LogOut size={18} />
          </span>
        </Tooltip>
      {/if}
    {/if}
    {/if}

    <!-- Collapse toggle -->
    <button
      onclick={() => collapsed = !collapsed}
      class="sidebar-item w-full justify-center text-text-muted px-3 py-2.5"
      title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
      aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
    >
      {#if collapsed}
        <ChevronRight size={16} />
      {:else}
        <ChevronLeft size={16} />
        <span class="text-xs">Collapse</span>
      {/if}
    </button>
  </div>
</aside>

<style>
  :global(.sidebar-item) {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    border-radius: 0.75rem;
    font-size: 0.875rem;
    line-height: 1.25rem;
    font-weight: 500;
    color: var(--color-text-secondary);
    border: 1px solid transparent;
    transition: background-color 0.2s, color 0.2s, border-color 0.2s, box-shadow 0.2s;
    transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
    cursor: pointer;
    user-select: none;
  }
  :global(.sidebar-item:hover) {
    background-color: color-mix(in srgb, var(--color-surface-hover) 50%, transparent);
    color: var(--color-text-primary);
    box-shadow: var(--shadow-glow-primary-sm);
  }

  :global(.sidebar-item-active) {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    border-radius: 0.75rem;
    font-size: 0.875rem;
    line-height: 1.25rem;
    font-weight: 500;
    color: var(--color-primary-light);
    transition: background-color 0.2s, color 0.2s, border-color 0.2s, box-shadow 0.2s;
    transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
    cursor: pointer;
    user-select: none;
    background-color: var(--color-primary-subtle);
    backdrop-filter: blur(24px);
    border: 1px solid color-mix(in srgb, var(--color-primary-default) 20%, transparent);
    box-shadow: var(--shadow-glow-primary);
    position: relative;
  }
  :global(.sidebar-item-active:hover) {
    background-color: color-mix(in srgb, var(--color-primary-default) 20%, transparent);
    color: white;
    border-color: color-mix(in srgb, var(--color-primary-default) 30%, transparent);
  }
  :global(.sidebar-item-active:active) {
    transform: scale(0.965) translateY(-1px);
  }

  :global(.sidebar-item-active::before) {
    content: '';
    position: absolute;
    left: 0;
    top: 25%;
    bottom: 25%;
    width: 3.5px;
    background: var(--color-primary-light);
    border-radius: 0 4px 4px 0;
    box-shadow: 0 0 10px rgba(139, 92, 246, 0.5);
    transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  }

  :global(.sidebar-item-active:hover::before) {
    top: 15%;
    bottom: 15%;
  }

  :global(.sidebar-parent-active) {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    border-radius: 0.75rem;
    padding: 0.625rem 0.75rem;
    font-size: 0.875rem;
    line-height: 1.25rem;
    font-weight: 500;
    transition: background-color 0.2s, color 0.2s;
    transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
    cursor: pointer;
    user-select: none;
    color: var(--color-primary-light);
    position: relative;
  }
  :global(.sidebar-parent-active:hover) {
    background-color: color-mix(in srgb, var(--color-surface-hover) 50%, transparent);
    box-shadow: var(--shadow-glow-primary-sm);
  }

  :global(.sidebar-parent-active::before) {
    content: '';
    position: absolute;
    left: 0;
    top: 35%;
    bottom: 35%;
    width: 2.5px;
    background: var(--color-primary-light);
    border-radius: 0 3px 3px 0;
    opacity: 0.5;
    transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  }

  :global(.sidebar-parent-active:hover::before) {
    opacity: 0.8;
    top: 30%;
    bottom: 30%;
  }

  :global(.sidebar-ripple) {
    position: absolute;
    border-radius: 50%;
    background: rgba(139, 92, 246, 0.3);
    transform: scale(0);
    animation: sidebar-ripple-anim 0.6s linear;
    pointer-events: none;
  }

  @keyframes sidebar-ripple-anim {
    to {
      transform: scale(4);
      opacity: 0;
    }
  }
</style>