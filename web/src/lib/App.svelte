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

  let Component = Home;
  let currentPath = getPath();

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
    currentPath = path;

    // Check authentication for all non-login routes
    if (path !== '/login') {
      const hasToken = sessionStorage.getItem('access_token') || localStorage.getItem('access_token');
      if (!hasToken) {
        // Not authenticated - redirect to login immediately
        Component = LoginPage;
        currentPath = '/login';
        window.history.replaceState({}, '', '/login');
        return;
      }
    }

    // Authenticated or on login page - set appropriate component
    if (path === '/login') {
      Component = LoginPage;
    } else if (path === '/') {
      Component = Home;
    } else {
      Component = getComponent(path);
    }
  }

  // Initial route handling with immediate auth check
  const initialPath = getPath();
  const hasToken = sessionStorage.getItem('access_token') || localStorage.getItem('access_token');

  if (initialPath !== '/login' && !hasToken) {
    // Trying to access protected route without auth - redirect immediately
    Component = LoginPage;
    currentPath = '/login';
    window.history.replaceState({}, '', '/login');
  } else {
    // Valid access - proceed with route
    handleRoute(initialPath);
  }

  // Subscribe to route changes
  subscribe(handleRoute);

  // Subscribe to route changes (popstate, etc.)
  subscribe(handleRoute);
</script>

{#if currentPath !== '/login'}
  <Navbar />
{/if}
<svelte:component this={Component} />
