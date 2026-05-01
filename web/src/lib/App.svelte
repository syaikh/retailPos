<script>
  import { goto, getPath, subscribe } from '$lib/router';
  import LoginPage from '$lib/pages/LoginPage.svelte';
  import Home from '$lib/pages/Home.svelte';
  import PosPage from '$lib/pages/PosPage.svelte';
  import InventoryPage from '$lib/pages/InventoryPage.svelte';
  import ReportsPage from '$lib/pages/ReportsPage.svelte';
  import AdminUsers from '$lib/pages/admin/Users.svelte';
  import AdminRoles from '$lib/pages/admin/Roles.svelte';
  import AdminAuditLogs from '$lib/pages/admin/AuditLogs.svelte';
  import Navbar from '$lib/components/Navbar.svelte';
  import { auth } from '$lib/stores/auth';

  let Component = LoginPage;
  let currentPath = getPath();
  let isInitializing = true;

  function getComponent(path) {
    switch (path) {
      case '/login': return LoginPage;
      case '/pos': return PosPage;
      case '/inventory': return InventoryPage;
      case '/reports': return ReportsPage;
      case '/admin': return AdminUsers;
      case '/admin/users': return AdminUsers;
      case '/admin/roles': return AdminRoles;
      case '/admin/audit-logs': return AdminAuditLogs;
      default: return Home;
    }
  }

  function handleRoute(path) {
    const hasToken = sessionStorage.getItem('access_token') || localStorage.getItem('access_token');

    // Check authentication for all non-login routes
    if (path !== '/login' && !hasToken) {
      // Not authenticated - force redirect to login
      Component = LoginPage;
      currentPath = '/login';
      window.history.replaceState({}, '', '/login');
      isInitializing = false;
      return;
    }

    // Authenticated or on login page - set appropriate component
    currentPath = path;
    isInitializing = false;
    if (path === '/login') {
      Component = LoginPage;
    } else if (path === '/') {
      Component = Home;
    } else {
      Component = getComponent(path);
    }
  }

  // Initial authentication check and route handling
  const initialPath = getPath();
  const hasToken = sessionStorage.getItem('access_token') || localStorage.getItem('access_token');

  if (initialPath !== '/login' && !hasToken) {
    // Accessing protected route without auth - redirect to login
    Component = LoginPage;
    currentPath = '/login';
    window.history.replaceState({}, '', '/login');
    isInitializing = false;
  } else {
    // Valid access - proceed normally
    handleRoute(initialPath);
  }

  // Subscribe to route changes (popstate, etc.)
  subscribe(handleRoute);
</script>

{#if isInitializing}
  <div class="min-h-screen bg-bg flex items-center justify-center">
    <div class="text-center">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto mb-4"></div>
      <p class="text-slate-400">Loading...</p>
    </div>
  </div>
{:else}
  {#if currentPath !== '/login'}
    <Navbar />
  {/if}
  <svelte:component this={Component} />
{/if}
