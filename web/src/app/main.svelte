<script>
  import { goto, getPath, subscribe } from '$app/router';
  import { restoreSession, useAuthStore } from '$modules/auth';
  import { initWebSocket } from '$app/providers/websocket';
  import { initAuth } from '$app/providers/auth-init';
  import ReceiptPrintOverlay from '$app/components/ReceiptPrintOverlay.svelte';
  import { fade } from 'svelte/transition';

  import Layout from '$app/layouts/Layout.svelte';
  import { Toast } from '$shared/ui';

  // Always-needed pages (loaded eagerly)
  import LoginPage from '$modules/auth/components/LoginPage.svelte';
  import Home from '$modules/dashboard/components/Home.svelte';

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
    '/categories':          'Category Management',
    '/admin':              'Administration',
    '/admin/users':        'User Management',
    '/admin/roles':        'Role Management',
    '/admin/audit-logs':   'Audit Logs',
    '/admin/categories':   'Category Management',
    '/admin/brands':       'Brand Management',
    '/admin/units-of-measure': 'Unit of Measure Management',
  };

  const pageModules = {
    '/pos':                 () => import('$modules/pos/components/PosPage.svelte'),
    '/inventory':           () => import('$modules/product/components/ProductsPage.svelte'),
    '/inventory/products':  () => import('$modules/product/components/ProductsPage.svelte'),
    '/reports':             () => import('$modules/reporting/components/ReportsPage.svelte'),
    '/transactions':        () => import('$modules/sales/components/TransactionsPage.svelte'),
    '/customers':           () => import('$modules/customers/components/CustomersPage.svelte'),
    '/categories':          () => import('$modules/settings/components/CategoriesPage.svelte'),
    '/admin':               () => import('$modules/admin/components/UsersPage.svelte'),
    '/admin/users':         () => import('$modules/admin/components/UsersPage.svelte'),
    '/admin/roles':         () => import('$modules/admin/components/RolesPage.svelte'),
    '/admin/audit-logs':    () => import('$modules/admin/components/AuditLogsPage.svelte'),
    '/admin/categories':    () => import('$modules/settings/components/CategoriesPage.svelte'),
    '/admin/brands':        () => import('$modules/settings/components/BrandsPage.svelte'),
    '/admin/units-of-measure': () => import('$modules/settings/components/UnitsOfMeasurePage.svelte'),
  };

  let loadId = 0;

  async function getComponent(path) {
    const loader = pageModules[path];
    if (!loader) return Home;
    const id = ++loadId;
    const mod = await loader();
    if (id !== loadId) return;
    return mod.default;
  }

  function updateTitle(path) {
    const page = pageTitles[path] || 'Dashboard';
    document.title = `${page} — RetailPOS`;
  }

  async function handleRoute(path) {
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

    if (hasValidToken && !wsInitialized) {
      wsInitialized = true;
      initWebSocket();
    }

    currentPath = path;
    isInitializing = false;
    const comp = await getComponent(path);
    if (comp) Component = comp;
    updateTitle(path);
  }

  let wsInitialized = false;

  async function initializeRoute(path) {
    await initAuth();
    const authStore = useAuthStore();
    console.log('[main] initAuth complete | isAuthenticated:', authStore.isAuthenticated, '| path:', path);

    if (!authStore.isAuthenticated) {
      console.log('[main] Not authenticated, redirecting to login');
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

    console.log('[main] Authenticated, calling initWebSocket');
    wsInitialized = true;
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
      const comp = await getComponent(path);
      if (comp) Component = comp;
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
