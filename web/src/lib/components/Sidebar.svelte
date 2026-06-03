<script lang="ts">
  import {
    LayoutDashboard,
    ShoppingCart,
    Package,
    BarChart3,
    Users,
    Shield,
    ScrollText,
    ChevronLeft,
    ChevronRight,
    LogOut,
    Store,
    User,
    Tag,
  } from 'lucide-svelte';
  import { goto, getPath } from '$lib/router';
  import { logout } from '$lib/api/auth';
  import { auth } from '$lib/stores/auth';

  let {
    currentPath = $bindable('/')
  }: { currentPath?: string } = $props();

  let collapsed = $state(false);
  let adminExpanded = $state(false);

  const isAdminPath = $derived(currentPath.startsWith('/admin'));

  // Auto-expand admin if on an admin route
  $effect(() => {
    if (isAdminPath) adminExpanded = true;
  });

  let username = $derived($auth.user?.username || 'User');
  let role = $derived($auth.user?.role?.name || ($auth.user?.role && typeof $auth.user?.role === 'object' ? $auth.user.role.name : $auth.user?.role) || ($auth.user?.role_id === 1 ? 'superadmin' : $auth.user?.role_id === 2 ? 'admin' : $auth.user?.role_id === 3 ? 'cashier' : $auth.user?.role_id === 4 ? 'manager' : 'cashier'));

  const navItems = [
    { label: 'Dashboard',  href: '/',           icon: LayoutDashboard },
    { label: 'Point of Sale', href: '/pos',     icon: ShoppingCart    },
    { label: 'Inventory',  href: '/inventory',  icon: Package         },
    { label: 'Reports',    href: '/reports',    icon: BarChart3       },
    { label: 'Categories', href: '/categories',  icon: Tag             },
  ];

  const adminItems = [
    { label: 'Users',       href: '/admin/users',       icon: Users      },
    { label: 'Roles',       href: '/admin/roles',       icon: Shield     },
    { label: 'Audit Logs',  href: '/admin/audit-logs',  icon: ScrollText },
  ];

  function isActive(href: string) {
    if (href === '/') return currentPath === '/';
    return currentPath === href;
  }

  function navigate(href: string) {
    goto(href);
  }

  async function handleLogout() {
    await logout();
    goto('/login');
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

<!-- Sidebar -->
<aside
  class="sidebar-shell flex flex-col bg-sidebar border-r border-sidebar-border shadow-sidebar shrink-0 transition-all duration-300 ease-spring"
  style:width={collapsed ? 'var(--sidebar-collapsed-width)' : 'var(--sidebar-width)'}
>
  <!-- Brand -->
  <div class="flex items-center gap-3 px-4 h-16 border-b border-sidebar-border">
    <div class="w-9 h-9 rounded-xl gradient-bg-primary flex items-center justify-center shrink-0 shadow-glow-primary-sm">
      <Store size={18} class="text-white" />
    </div>
    {#if !collapsed}
      <div class="animate-fade-in overflow-hidden">
        <p class="text-sm font-bold text-text-primary leading-tight truncate">RetailPOS</p>
        <p class="text-[10px] text-text-muted truncate">Management System</p>
      </div>
    {/if}
  </div>

  <!-- Nav -->
  <nav class="flex-1 overflow-y-auto overflow-x-hidden py-4 px-2.5 space-y-0.5 no-scrollbar">
    {#each navItems as item}
      <button
        onclick={(e) => { createRipple(e, e.currentTarget); navigate(item.href); }}
        class={isActive(item.href) ? 'sidebar-item-active w-full text-left relative overflow-hidden' : 'sidebar-item w-full text-left relative overflow-hidden'}
        title={collapsed ? item.label : ''}
      >
        <item.icon size={18} class="shrink-0" />
        {#if !collapsed}
          <span class="animate-fade-in relative z-10">{item.label}</span>
        {/if}
      </button>
    {/each}

    <!-- Admin section -->
    <div class="pt-4 pb-1">
      {#if !collapsed}
        <p class="px-3 text-[10px] font-semibold uppercase tracking-widest text-text-muted mb-1 animate-fade-in">
          Administration
        </p>
      {:else}
        <div class="border-t border-sidebar-border mx-2 mb-2"></div>
      {/if}
    </div>

    {#each adminItems as item}
      <button
        onclick={(e) => { createRipple(e, e.currentTarget); navigate(item.href); }}
        class={isActive(item.href) ? 'sidebar-item-active w-full text-left relative overflow-hidden' : 'sidebar-item w-full text-left relative overflow-hidden'}
        title={collapsed ? item.label : ''}
      >
        <item.icon size={18} class="shrink-0" />
        {#if !collapsed}
          <span class="animate-fade-in relative z-10">{item.label}</span>
        {/if}
      </button>
    {/each}
  </nav>

  <!-- Bottom: user + collapse toggle -->
  <div class="mt-auto border-t border-sidebar-border px-2.5 py-3 space-y-0.5">
    <!-- User row -->
    <div class="flex items-center gap-3 px-3 py-2.5 rounded-xl" title={collapsed ? username : ''}>
      <div class="w-8 h-8 rounded-full gradient-bg-primary flex items-center justify-center shrink-0">
        <User size={14} class="text-white" />
      </div>
      {#if !collapsed}
        <div class="flex-1 min-w-0 animate-fade-in">
          <p class="text-xs font-semibold text-text-primary truncate">{username}</p>
          <p class="text-[10px] text-text-muted capitalize truncate">{role}</p>
        </div>
        <button
          onclick={handleLogout}
          class="flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-text-muted hover:text-danger hover:bg-danger-subtle transition-all duration-200 group"
          title="Logout"
        >
          <LogOut size={14} class="group-hover:scale-110 transition-transform" />
          <span class="text-xs font-medium">Logout</span>
        </button>
      {/if}
    </div>

    {#if collapsed}
      <button
        onclick={handleLogout}
        class="sidebar-item w-full justify-center text-text-muted hover:text-danger hover:bg-danger-subtle"
        title="Logout"
      >
        <LogOut size={18} />
      </button>
    {/if}

    <!-- Collapse toggle -->
    <button
      onclick={() => collapsed = !collapsed}
      class="sidebar-item w-full justify-center text-text-muted"
      title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
    >
      {#if collapsed}
        <ChevronRight size={16} />
      {:else}
        <ChevronLeft size={16} />
        <span class="text-xs animate-fade-in">Collapse</span>
      {/if}
    </button>
  </div>
</aside>
