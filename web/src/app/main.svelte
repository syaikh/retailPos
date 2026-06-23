<script>
  import { goto, getPath, subscribe } from '$app/router';
  import { restoreSession, useAuthStore } from '$modules/auth';
  import { initWebSocket } from '$app/providers/websocket';
  import { initAuth } from '$app/providers/auth-init';
  import ReceiptPrintOverlay from '$app/components/ReceiptPrintOverlay.svelte';
  import { fade } from 'svelte/transition';

  import LoginPage from '$modules/auth/components/LoginPage.svelte';
  import Home from '$modules/dashboard/components/Home.svelte';
  import PosPage from '$modules/pos/components/PosPage.svelte';
  import ProductsPage from '$modules/product/components/ProductsPage.svelte';
  import ReportsPage from '$modules/reporting/components/ReportsPage.svelte';
  import TransactionsPage from '$modules/sales/components/TransactionsPage.svelte';
  import CustomersPage from '$modules/customers/components/CustomersPage.svelte';
  import UsersPage from '$modules/admin/components/UsersPage.svelte';
  import RolesPage from '$modules/admin/components/RolesPage.svelte';
  import AuditLogsPage from '$modules/admin/components/AuditLogsPage.svelte';
  import CategoriesPage from '$modules/settings/components/CategoriesPage.svelte';
  import BrandsPage from '$modules/settings/components/BrandsPage.svelte';
  import UnitsOfMeasurePage from '$modules/settings/components/UnitsOfMeasurePage.svelte';
  import Layout from '$app/layouts/Layout.svelte';
  import { Toast } from '$shared/ui';

  let Component = $state(LoginPage);
  let currentPath = $state(getPath());
  let isInitializing = $state(true);

  const pageTitles = {
    '/login':              'Login',
    '/':                   'Dashboard',
    '/pos':                'Point of Sale',
    '/inventory':          'Products',
    '/inventory/products': 'Products',
    '/reports':            'Reports',
    '/transactions':       'Transaction History',
    '/customers':          'Customers',
    '/admin':              'Administration',
    '/admin/users':        'User Management',
    '/admin/roles':        'Role Management',
    '/admin/audit-logs':   'Audit Logs',
    '/admin/categories':   'Category Management',
    '/admin/brands':       'Brand Management',
    '/admin/units-of-measure': 'Unit of Measure Management',
  };

  function getComponent(path) {
    switch (path) {
      case '/login':               return LoginPage;
      case '/pos':                 return PosPage;
      case '/inventory':           return ProductsPage;
      case '/inventory/products':  return ProductsPage;
      case '/reports':             return ReportsPage;
      case '/transactions':        return TransactionsPage;
      case '/customers':           return CustomersPage;
      case '/admin':               return UsersPage;
      case '/admin/users':         return UsersPage;
      case '/admin/roles':         return RolesPage;
      case '/admin/audit-logs':    return AuditLogsPage;
      case '/admin/categories':    return CategoriesPage;
      case '/admin/brands':        return BrandsPage;
      case '/admin/units-of-measure': return UnitsOfMeasurePage;
      default:                     return Home;
    }
  }

  function updateTitle(path) {
    const page = pageTitles[path] || 'Dashboard';
    document.title = `${page} — RetailPOS`;
  }

  function handleRoute(path) {
    const token = sessionStorage.getItem('access_token');
    const hasValidToken = token && token !== 'null' && token !== 'undefined' && token.length > 10;

    if (path !== '/login' && !hasValidToken) {
      Component = LoginPage;
      currentPath = '/login';
      window.history.replaceState({}, '', '/login');
      isInitializing = false;
      updateTitle('/login');
      return;
    }

    if (path === '/login' && hasValidToken) {
      goto('/');
      return;
    }

    currentPath = path;
    isInitializing = false;
    Component = getComponent(path);
    updateTitle(path);
  }

  async function initializeRoute(path) {
    await initAuth();
    const authStore = useAuthStore();

    if (!authStore.isAuthenticated) {
      if (path !== '/login') {
        Component = LoginPage;
        currentPath = '/login';
        window.history.replaceState({}, '', '/login');
      } else {
        Component = LoginPage;
        currentPath = '/login';
      }
      updateTitle('/login');
      isInitializing = false;
      subscribe(handleRoute);
      return;
    }

    initWebSocket();
    subscribe(handleRoute);

    if (path === '/login') {
      Component = Home;
      currentPath = '/';
      window.history.replaceState({}, '', '/');
      updateTitle('/');
    } else {
      if (path === '/inventory') {
        goto('/inventory/products');
        isInitializing = false;
        return;
      }
      if (path === '/inventory/stock') {
        goto('/inventory/products');
        isInitializing = false;
        return;
      }

      currentPath = path;
      Component = getComponent(path);
      updateTitle(path);
    }

    isInitializing = false;
    handleRoute(getPath());
  }

  const initialPath = getPath();
  initializeRoute(initialPath);
</script>

{#if isInitializing}
  <div class="min-h-screen bg-bg flex items-center justify-center absolute inset-0 z-50" out:fade={{ duration: 300 }}>
    <div class="flex flex-col items-center gap-4">
      <div class="w-12 h-12 rounded-2xl gradient-bg-primary flex items-center justify-center shadow-glow-primary animate-pulse">
        <span class="text-white text-xl font-bold">R</span>
      </div>
      <div class="flex gap-1.5">
        <span class="w-2 h-2 bg-primary rounded-full animate-bounce" style="animation-delay:0ms"></span>
        <span class="w-2 h-2 bg-primary rounded-full animate-bounce" style="animation-delay:150ms"></span>
        <span class="w-2 h-2 bg-primary rounded-full animate-bounce" style="animation-delay:300ms"></span>
      </div>
    </div>
  </div>
{:else if currentPath === '/login'}
  <Component />
{:else}
  <Layout {currentPath}>
    <Component />
  </Layout>
{/if}

<Toast />

<ReceiptPrintOverlay />
