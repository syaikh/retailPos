<script>
  import { page } from '$app/stores';
  import { auth } from '$lib/stores/auth';
  import { logout as apiLogout } from '$lib/api/auth';
  import { cart } from '$lib/stores.js';
  import { 
    LayoutDashboard, 
    ShoppingCart, 
    Package, 
    Tags,
    BarChart3, 
    LogOut,
    Store
  } from 'lucide-svelte';

  let role = $derived($auth.user?.role || 'cashier');

  const menuItems = [
    { name: 'Dashboard', icon: LayoutDashboard, path: '/', roles: ['admin'] },
    { name: 'POS / Kasir', icon: ShoppingCart, path: '/pos', roles: ['admin', 'cashier'] },
    { name: 'Inventory', icon: Package, path: '/inventory', roles: ['admin'] },
    { name: 'Kategori', icon: Tags, path: '/inventory/groups', roles: ['admin'] },
    { name: 'Laporan', icon: BarChart3, path: '/reports', roles: ['admin'] },
  ];

  let activePath = $derived($page.url.pathname);

  function handleLogout() {
    cart.set([]);
    apiLogout();
  }
</script>

<aside class="sidebar">
  <div class="logo">
    <Store size={28} color="var(--primary)" />
    <span>RetailPOS</span>
  </div>

  <nav class="nav-links">
    {#each menuItems as item}
      {#if item.roles.includes(role)}
        <a
          class="nav-item"
          class:active={activePath === item.path}
          href={item.path}
        >
          <item.icon size={20} />
          <span>{item.name}</span>
        </a>
      {/if}
    {/each}
  </nav>

  <div class="footer">
    <button class="logout-btn" onclick={handleLogout}>
      <LogOut size={20} />
      <span>Logout</span>
    </button>
  </div>
</aside>

<style>
  .sidebar {
    width: var(--sidebar-width);
    background: var(--bg-surface);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    padding: 24px 0;
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 0 24px 32px;
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text-primary);
    border-bottom: 1px solid var(--border);
    margin-bottom: 24px;
  }

  .nav-links {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 0 12px;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    background: transparent;
    border: none;
    color: var(--text-secondary);
    text-align: left;
    width: 100%;
    cursor: pointer;
    border-radius: 8px;
    transition: all 0.2s ease;
  }

  .nav-item:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .nav-item.active {
    background: var(--primary);
    color: white;
  }

  .nav-item:hover {
    background: rgba(99, 102, 241, 0.1);
    color: var(--primary);
  }

  .nav-item.active {
    background: var(--primary);
    color: white;
  }

  .footer {
    padding: 0 12px;
    margin-top: auto;
  }

  .logout-btn {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    width: 100%;
    color: var(--danger);
    background: transparent;
    text-align: left;
  }

  .logout-btn:hover {
    background: rgba(239, 68, 68, 0.1);
  }
</style>
