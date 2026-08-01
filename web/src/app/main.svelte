<script>
  import { goto, getPath, subscribe } from '$app/router';
  import { restoreSession, useAuthStore, startProactiveRefresh } from '$modules/auth';
  import { initWebSocket } from '$app/providers/websocket';
  import { initAuth } from '$app/providers/auth-init';
  import ReceiptPrintOverlay from '$app/components/ReceiptPrintOverlay.svelte';
  import NotFoundPage from '$app/components/NotFoundPage.svelte';
  import { fade } from 'svelte/transition';
  import { routePermissions } from '$app/config/permissions';
  import { getDefaultRoute } from '$shared/utils/default-route';
  import { hasPermission } from '$shared/utils/permissions';
  import { toast } from '$shared/stores/toast.svelte';

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
    '/categories/import-history': 'Import History — Categories',
    '/brands/import-history':     'Import History — Brands',
    '/units-of-measure/import-history': 'Import History — Units',
    '/customers/import-history':  'Import History — Customers',
    '/products/import-history':   'Import History — Products',
    '/admin':              'Administration',
    '/admin/users':        'User Management',
    '/admin/roles':        'Role Management',
    '/admin/audit-logs':   'Audit Logs',
    '/admin/categories':   'Category Management',
    '/stores':             'Store Management',
    '/stores/import-history': 'Import History — Stores',
    '/brands':             'Brand Management',
    '/units-of-measure':   'Unit of Measure Management',
    '/pricing-rules':      'Pricing Rules',
    '/customer-groups':    'Customer Groups',
    '/suppliers':          'Supplier Management',
    '/admin/brands':       'Brand Management',
    '/admin/units-of-measure': 'Unit of Measure Management',
    '/shifts':             'Shift Management',
    '/purchase-orders':    'Purchase Orders',
  };

  const pageModules = {
    '/':                   () => Promise.resolve({ default: Home }),
    '/pos':                () => import('$modules/pos/components/PosPage.svelte'),
    '/inventory':           () => import('$modules/product/components/ProductsPage.svelte'),
    '/inventory/products':  () => import('$modules/product/components/ProductsPage.svelte'),
    '/reports':             () => import('$modules/reporting/components/ReportsPage.svelte'),
    '/transactions':        () => import('$modules/sales/components/TransactionsPage.svelte'),
    '/customers':           () => import('$modules/customers/components/CustomersPage.svelte'),
    '/categories':          () => import('$modules/settings/components/CategoriesPage.svelte'),
    '/categories/import-history':  () => import('$modules/import-export/components/ImportHistoryPage.svelte'),
    '/brands/import-history':      () => import('$modules/import-export/components/ImportHistoryPage.svelte'),
    '/units-of-measure/import-history': () => import('$modules/import-export/components/ImportHistoryPage.svelte'),
    '/customers/import-history':   () => import('$modules/import-export/components/ImportHistoryPage.svelte'),
    '/products/import-history':    () => import('$modules/import-export/components/ImportHistoryPage.svelte'),
    '/admin':               () => import('$modules/admin/components/UsersPage.svelte'),
    '/admin/users':         () => import('$modules/admin/components/UsersPage.svelte'),
    '/admin/roles':         () => import('$modules/admin/components/RolesPage.svelte'),
    '/admin/audit-logs':    () => import('$modules/admin/components/AuditLogsPage.svelte'),
    '/stores':              () => import('$modules/stores/components/StoresPage.svelte'),
    '/stores/import-history': () => import('$modules/import-export/components/ImportHistoryPage.svelte'),
    '/admin/categories':    () => import('$modules/settings/components/CategoriesPage.svelte'),
    '/brands':              () => import('$modules/settings/components/BrandsPage.svelte'),
    '/units-of-measure':    () => import('$modules/settings/components/UnitsOfMeasurePage.svelte'),
    '/pricing-rules':       () => import('$modules/pricing/components/PricingRulesPage.svelte'),
    '/customer-groups':     () => import('$modules/customer-groups/components/CustomerGroupsPage.svelte'),
    '/suppliers':           () => import('$modules/supplier/components/SuppliersPage.svelte'),
    '/admin/brands':        () => import('$modules/settings/components/BrandsPage.svelte'),
    '/admin/units-of-measure': () => import('$modules/settings/components/UnitsOfMeasurePage.svelte'),
    '/shifts':              () => import('$modules/shifts/components/ShiftsPage.svelte'),
    '/purchase-orders':     () => import('$modules/purchase-orders/components/PurchaseOrdersPage.svelte'),
  };

  let loadId = 0;

  async function getComponent(path) {
    const loader = pageModules[path];
    if (!loader) return NotFoundPage;
    const id = ++loadId;
    try {
      const mod = await loader();
      if (id !== loadId) return;
      return mod.default;
    } catch (err) {
      console.error('Failed to load page:', err);
      return NotFoundPage;
    }
  }

  function updateTitle(path) {
    const page = pageTitles[path] || 'Dashboard';
    document.title = `${page} — RetailPOS`;
  }

  function hasRoutePermission(path) {
    const requiredPerms = routePermissions[path];
    if (!requiredPerms) return true;
    const authStore = useAuthStore();
    const userPerms = authStore.user?.permissions || [];
    return requiredPerms.some(p => hasPermission(userPerms, p));
  }

  async function handleRoute(fullPath) {
    const path = fullPath.split('?')[0];
    const token = sessionStorage.getItem('access_token');
    const hasValidToken = token && token !== 'null' && token !== 'undefined' && token.length > 10;

    if (!hasValidToken) {
      Component = LoginPage;
      currentPath = '/login';
      if (path !== '/login') {
        window.history.replaceState({}, '', '/login');
      }
      isInitializing = false;
      updateTitle('/login');
      return;
    }

    if (path === '/login') {
      const defaultRoute = getDefaultRoute(useAuthStore().user);
      goto(defaultRoute);
      return;
    }

    if (!hasRoutePermission(path)) {
      toast.error('You do not have permission to access this page');
      const fallback = getDefaultRoute(useAuthStore().user);
      if (fallback === path) {
        goto('/');
      } else {
        goto(fallback);
      }
      return;
    }

    if (hasValidToken && !wsInitialized) {
      wsInitialized = true;
      initWebSocket();
    }

    const comp = await getComponent(path);
    if (comp) Component = comp;
    currentPath = path;
    isInitializing = false;
    updateTitle(path);
  }

  let wsInitialized = false;

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

    wsInitialized = true;
    initWebSocket();
    startProactiveRefresh();
    subscribe(handleRoute);

    if (path === '/login') {
      const defaultRoute = getDefaultRoute(authStore.user);
      currentPath = defaultRoute;
      window.history.replaceState({}, '', defaultRoute);
      const comp = await getComponent(defaultRoute);
      if (comp) Component = comp;
      updateTitle(defaultRoute);
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

      if (!hasRoutePermission(path)) {
        toast.error('You do not have permission to access this page');
        let fallback = getDefaultRoute(authStore.user);
        if (fallback === path) {
          fallback = '/';
        }
        const comp = await getComponent(fallback);
        if (comp) Component = comp;
        currentPath = fallback;
        window.history.replaceState({}, '', fallback);
        updateTitle(fallback);
        isInitializing = false;
        return;
      }

      currentPath = path;
      const comp = await getComponent(path);
      if (comp) Component = comp;
      updateTitle(path);
    }

    isInitializing = false;
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
