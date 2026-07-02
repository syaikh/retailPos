<script lang="ts">
  import { onMount } from 'svelte';
  import apiClient from '$shared/api/http-client';
  import { toast } from '$shared/stores/toast.svelte';
  import { debounce } from '$shared/utils/debounce';
  import { useAuthStore } from '$modules/auth';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, JAKARTA_OFFSET_MS } from '$shared/utils/jakartaTime';
  import { ScrollText } from 'lucide-svelte';
  import AuditLogsFilterToolbar from './AuditLogsFilterToolbar.svelte';
  import AuditLogsTable from './AuditLogsTable.svelte';
  import AuditLogDetailsDrawer from './AuditLogDetailsDrawer.svelte';

  const authStore = useAuthStore();

  // State variables
  let loading = $state(true);
  let items = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let selectedAction = $state('all');
  let selectedResource = $state('all');
  let selectedDateRange = $state('24h');
  let showDatePicker = $state(false);

  let customStartDate = $state(getDateNDaysAgoInJakarta(1));
  let customEndDate = $state(getTodayInJakarta());

  // Request tracking to prevent duplicate requests
  let currentRequestId = $state(0);
  let abortController = $state(null);
  let hasInitialized = $state(false);

  let userRole = $derived(
    authStore.user?.role?.name ||
      (authStore.user?.role && typeof authStore.user?.role === 'object' ? authStore.user.role.name : authStore.user?.role) ||
      ''
  );
  let canView = $derived(userRole === 'superadmin');

  // Drawer state
  let drawerOpen = $state(false);
  let selectedLog = $state(null);

  function openDrawer(log) {
    selectedLog = log;
    drawerOpen = true;
  }

  function closeDrawer() {
    drawerOpen = false;
    selectedLog = null;
  }

  // Convert a Jakarta date string (YYYY-MM-DD) to UTC epoch for RFC3339 API
  function jakartaDateToUTC(dateStr) {
    const [y, m, d] = dateStr.split('-').map(Number);
    // Jakarta midnight = UTC 17:00 previous day
    return Date.UTC(y, m - 1, d, 0, 0, 0, 0) - JAKARTA_OFFSET_MS;
  }

  function getJakartaMidnightMs(dateStr: string): number {
    const [y, m, d] = dateStr.split('-').map(Number);
    return Date.UTC(y, m - 1, d, 0, 0, 0, 0) - JAKARTA_OFFSET_MS;
  }

  function getDateRange(range) {
    switch (range) {
      case '24h': {
        const yesterday = getDateNDaysAgoInJakarta(1);
        const todayJakarta = getTodayInJakarta();
        return {
          start: new Date(getJakartaMidnightMs(yesterday)),
          end: new Date(getJakartaMidnightMs(todayJakarta) + 86400000),
        };
      }
      case '7d': {
        const sevenDaysAgo = getDateNDaysAgoInJakarta(7);
        const todayJakarta = getTodayInJakarta();
        return {
          start: new Date(getJakartaMidnightMs(sevenDaysAgo)),
          end: new Date(getJakartaMidnightMs(todayJakarta) + 86400000),
        };
      }
      case '30d': {
        const thirtyDaysAgo = getDateNDaysAgoInJakarta(30);
        const todayJakarta = getTodayInJakarta();
        return {
          start: new Date(getJakartaMidnightMs(thirtyDaysAgo)),
          end: new Date(getJakartaMidnightMs(todayJakarta) + 86400000),
        };
      }
      case '90d': {
        const ninetyDaysAgo = getDateNDaysAgoInJakarta(90);
        const todayJakarta = getTodayInJakarta();
        return {
          start: new Date(getJakartaMidnightMs(ninetyDaysAgo)),
          end: new Date(getJakartaMidnightMs(todayJakarta) + 86400000),
        };
      }
      case 'custom':
        if (customStartDate && customEndDate) {
          const startMs = jakartaDateToUTC(customStartDate);
          const endMs = jakartaDateToUTC(customEndDate) + 86400000;
          return { start: new Date(startMs), end: new Date(endMs) };
        }
        const fallbackStart = getDateNDaysAgoInJakarta(7);
        const fallbackEnd = getTodayInJakarta();
        return {
          start: new Date(getJakartaMidnightMs(fallbackStart)),
          end: new Date(getJakartaMidnightMs(fallbackEnd) + 86400000),
        };
      default: {
        const defaultStart = getDateNDaysAgoInJakarta(7);
        const defaultEnd = getTodayInJakarta();
        return {
          start: new Date(getJakartaMidnightMs(defaultStart)),
          end: new Date(getJakartaMidnightMs(defaultEnd) + 86400000),
        };
      }
    }
  }

  // --- Data fetching ---

  async function fetchLogs() {
    if (abortController) abortController.abort();

    const requestId = ++currentRequestId;
    abortController = new AbortController();

    try {
      loading = true;
      const range = getDateRange(selectedDateRange);
      const params = new URLSearchParams({
        limit: limit.toString(),
        offset: offset.toString(),
        search: searchQuery,
        start_date: range.start.toISOString(),
        end_date: range.end.toISOString(),
      });
      if (selectedAction !== 'all') params.append('action', selectedAction);
      if (selectedResource !== 'all') params.append('entity_type', selectedResource);

      const response = await apiClient.get(`audit-logs?${params.toString()}`, {
        signal: abortController.signal,
      });

      if (requestId !== currentRequestId) return;

      const data = response.data || {};
      items = data.data || [];
      total = data.total || 0;
    } catch (error) {
      const isCanceled =
        error?.name === 'CanceledError' ||
        error?.name === 'AbortError' ||
        error?.code === 'ERR_CANCELED' ||
        /canceled/i.test(error?.message || '');
      if (!isCanceled) {
        console.error('[AuditLogs] fetch error:', error);
        console.error('[AuditLogs] error.response:', error?.response);
        console.error('[AuditLogs] error.request:', error?.request);
        const msg = error?.response?.data?.error || error?.response?.data || error?.message || 'Unknown error';
        toast.error(`Failed to load audit logs: ${msg}`);
      }
    } finally {
      if (requestId === currentRequestId) {
        loading = false;
        abortController = null;
      }
    }
  }

  let prevSearch = $state('');
  let prevAction = $state('all');
  let prevEntity = $state('all');
  let prevDate = $state('24h');
  let prevOff = $state(0);
  let prevLim = $state(20);

  const debouncedSearchFetch = debounce(() => {
    offset = 0;
    fetchLogs();
  }, 400);

  $effect(() => {
    if (canView && !hasInitialized) {
      hasInitialized = true;
      prevSearch = searchQuery;
      prevAction = selectedAction;
      prevEntity = selectedResource;
      prevDate = selectedDateRange;
      prevOff = offset;
      prevLim = limit;
      fetchLogs();
    }
  });

  $effect(() => {
    if (!hasInitialized) return;

    const sq = searchQuery,
      sa = selectedAction,
      se = selectedResource,
      sd = selectedDateRange,
      so = offset,
      sl = limit;

    const searchChanged = sq !== prevSearch;
    const filterChanged = sa !== prevAction || se !== prevEntity || sd !== prevDate;
    const pageChanged = so !== prevOff || sl !== prevLim;

    if (searchChanged) debouncedSearchFetch();
    else if (filterChanged) { offset = 0; fetchLogs(); }
    else if (pageChanged) fetchLogs();

    prevSearch = sq;
    prevAction = sa;
    prevEntity = se;
    prevDate = sd;
    prevOff = so;
    prevLim = sl;
  });

  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
  }

  // Close date picker on outside click / Esc
  onMount(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (showDatePicker) {
        const target = e.target as HTMLElement;
        if (!target.closest('.date-picker-container') && !target.closest('.date-picker-trigger')) showDatePicker = false;
      }
    };
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (drawerOpen) closeDrawer();
        if (showDatePicker) showDatePicker = false;
      }
    };

    document.addEventListener('click', handleClickOutside);
    document.addEventListener('keydown', handleEsc);
    return () => {
      document.removeEventListener('click', handleClickOutside);
      document.removeEventListener('keydown', handleEsc);
    };
  });
</script>

{#if !canView}
  <div class="card px-4 py-16 text-center">
    <ScrollText size={40} class="text-text-muted mx-auto mb-4" />
    <p class="text-text-primary font-semibold text-lg">Access Denied</p>
    <p class="text-text-muted text-sm mt-1">Audit logs are restricted to superadmin only</p>
  </div>
{:else}
  <div class="space-y-5 max-w-7xl mx-auto">
    <AuditLogsFilterToolbar
      bind:searchQuery
      bind:selectedAction
      bind:selectedResource
      bind:selectedDateRange
      bind:showDatePicker
      bind:customStartDate
      bind:customEndDate
      {loading}
      onrefresh={() => { offset = 0; fetchLogs(); }}
    />
    <AuditLogsTable
      {items}
      {loading}
      {total}
      {limit}
      {offset}
      onpagechange={handlePageChange}
      onrowclick={openDrawer}
    />
    <AuditLogDetailsDrawer
      bind:drawerOpen
      selectedLog={selectedLog}
      onclose={closeDrawer}
    />
  </div>
{/if}


