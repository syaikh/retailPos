<script lang="ts">
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import apiClient from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { debounce } from '$lib/utils/debounce';
  import { auth, getAuthToken } from '$lib/stores/auth';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, JAKARTA_OFFSET_MS, formatDateInJakarta, formatTimeInJakarta, formatDateTimeInJakarta } from '$lib/utils/jakartaTime';

  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import ActionBadge from '$lib/components/ui/ActionBadge.svelte';
  import SearchBar from '$lib/components/ui/SearchBar.svelte';
  import {
    Search, ScrollText, RefreshCw, X, Download,
    Plus, Edit, Trash, LogIn, LogOut, Info, FileSpreadsheet, ArrowRight, Minus,
    Monitor, Globe, Package, ShoppingCart,
    Users, Tag, Shield, Store, Calendar, ChevronDown, List, CalendarDays
  } from 'lucide-svelte';

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
  let showResourceDropdown = $state(false);
  let showActionDropdown = $state(false);
  let showExportDropdown = $state(false);

  // Request tracking to prevent duplicate requests
  let currentRequestId = $state(0);
  let abortController = $state(null);
  let hasInitialized = $state(false);

  let userRole = $derived(
    $auth.user?.role?.name ||
      ($auth.user?.role && typeof $auth.user?.role === 'object' ? $auth.user.role.name : $auth.user?.role) ||
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

  // --- Filter definitions ---

  const actionsMap: Record<string, string[]> = {
    all: ['all', 'create', 'update', 'delete', 'login', 'logout'],
    auth: ['all', 'login', 'logout'],
    user: ['all', 'create', 'update', 'delete'],
    role: ['all', 'create', 'update', 'delete'],
    product: ['all', 'create', 'update', 'delete'],
    sale: ['all', 'create', 'update', 'delete'],
    category: ['all', 'create', 'update', 'delete'],
    brand: ['all', 'create', 'update', 'delete'],
  };

  const ALL_ACTIONS = [
    { id: 'all', label: 'All Actions' },
    { id: 'create', label: 'Create' },
    { id: 'update', label: 'Update' },
    { id: 'delete', label: 'Delete' },
    { id: 'login', label: 'Login' },
    { id: 'logout', label: 'Logout' },
  ];

  let availableActionFilters = $derived(
    selectedResource === 'all' ? ALL_ACTIONS : (actionsMap[selectedResource] || ['all']).map((id) => ALL_ACTIONS.find((a) => a.id === id) || { id, label: id })
  );

  const actionFilters = [
    { id: 'all', label: 'All', icon: List },
    { id: 'create', label: 'Create', icon: Plus },
    { id: 'update', label: 'Update', icon: Edit },
    { id: 'delete', label: 'Delete', icon: Trash },
    { id: 'login', label: 'Login', icon: LogIn },
    { id: 'logout', label: 'Logout', icon: LogOut },
  ];

  function isActionDisabled(actionId: string): boolean {
    if (selectedResource === 'all') return false;
    if (actionId === 'all') return false;
    const relevant = resourceActionMap[selectedResource];
    if (!relevant) return true;
    return !relevant.includes(actionId);
  }

  const resourceFilters = [
    { id: 'all', label: 'All Resources', icon: List },
    { id: 'auth', label: 'Auth', icon: Shield },
    { id: 'user', label: 'User', icon: Users },
    { id: 'role', label: 'Role', icon: Shield },
    { id: 'product', label: 'Product', icon: Package },
    { id: 'sale', label: 'Sale', icon: ShoppingCart },
    { id: 'category', label: 'Category', icon: Tag },
    { id: 'brand', label: 'Brand', icon: Store },
  ];

  const dateRanges = [
    { id: '24h', label: 'Last 24 Hours' },
    { id: '7d', label: 'Last 7 Days' },
    { id: '30d', label: 'Last 30 Days' },
    { id: '90d', label: 'Last 90 Days' },
    { id: 'custom', label: 'Custom Range' },
  ];

  const datePresets = [
    { label: 'Last 24 Hours', rangeId: '24h' },
    { label: 'Last 7 Days', rangeId: '7d' },
    { label: 'Last 30 Days', rangeId: '30d' },
    { label: 'Last 90 Days', rangeId: '90d' },
  ];

  const dateRangeLabel = $derived.by(() => {
    if (selectedDateRange === 'custom') {
      return `${customStartDate} – ${customEndDate}`;
    }
    return dateRanges.find(d => d.id === selectedDateRange)?.label || 'Last 24 Hours';
  });

  let customStartDate = $state('');
  let customEndDate = $state('');

  let resourceLabel = $derived(resourceFilters.find(f => f.id === selectedResource)?.label || 'All');
  let actionLabel = $derived(availableActionFilters.find(f => f.id === selectedAction)?.label || 'All Actions');

  const today = getTodayInJakarta();
  const ninetyDaysAgo = getDateNDaysAgoInJakarta(90);

  // Convert a Jakarta date string (YYYY-MM-DD) to UTC epoch for RFC3339 API
  function jakartaDateToUTC(dateStr) {
    const [y, m, d] = dateStr.split('-').map(Number);
    // Jakarta midnight = UTC 17:00 previous day
    // Date.UTC(y, m-1, d, 0,0,0,0) gives UTC midnight of the date
    return Date.UTC(y, m - 1, d, 0, 0, 0, 0) - JAKARTA_OFFSET_MS;
  }

  function getJakartaMidnightMs(dateStr: string): number {
    const [y, m, d] = dateStr.split('-').map(Number);
    return Date.UTC(y, m - 1, d, 0, 0, 0, 0) - JAKARTA_OFFSET_MS;
  }

  function getDateRange(range) {
    switch (range) {
      case '24h':
        return { start: new Date(Date.now() - 86400000), end: new Date(Date.now()) };
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
          // Interpret user-selected dates as Jakarta dates, convert to UTC epoch
          const startMs = jakartaDateToUTC(customStartDate);
          // End date inclusive: midnight Jakarta of next day
          const endMs = jakartaDateToUTC(customEndDate) + 86400000;
          return { start: new Date(startMs), end: new Date(endMs) };
        }
        return { start: new Date(Date.now() - 7 * 86400000), end: new Date() };
      default:
        return { start: new Date(Date.now() - 7 * 86400000), end: new Date() };
    }
  }

  function applyDatePreset(rangeId: string) {
    selectedDateRange = rangeId;
    showDatePicker = false;
    offset = 0;
    fetchLogs();
  }

  function applyCustomDateRange() {
    if (!customStartDate || !customEndDate) return;
    showDatePicker = false;
    selectedDateRange = 'custom';
    offset = 0;
    fetchLogs();
  }

  function clearDateFilter() {
    selectedDateRange = '24h';
    offset = 0;
    fetchLogs();
  }

  // Close dropdowns on outside click / Esc
  onMount(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (showDatePicker) {
        const target = e.target as HTMLElement;
        if (!target.closest('.date-picker-container') && !target.closest('.date-picker-trigger')) showDatePicker = false;
      }
      const resourceContainer = document.getElementById('resource-dropdown-container');
      if (showResourceDropdown && resourceContainer && !resourceContainer.contains(e.target as Node)) showResourceDropdown = false;
      const actionContainer = document.getElementById('action-dropdown-container');
      if (showActionDropdown && actionContainer && !actionContainer.contains(e.target as Node)) showActionDropdown = false;
      if (showExportDropdown) {
        const path = (e.composedPath?.() || []) as HTMLElement[];
        const inExport = path.some(
          (el) => el?.classList?.contains('export-dropdown') || el?.closest?.('.export-dropdown')
        );
        if (!inExport) showExportDropdown = false;
      }
    };
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (drawerOpen) closeDrawer();
        if (showDatePicker) showDatePicker = false;
        if (showResourceDropdown) showResourceDropdown = false;
        if (showActionDropdown) showActionDropdown = false;
        if (showExportDropdown) showExportDropdown = false;
      }
    };

    document.addEventListener('click', handleClickOutside);
    document.addEventListener('keydown', handleEsc);
    return () => {
      document.removeEventListener('click', handleClickOutside);
      document.removeEventListener('keydown', handleEsc);
    };
  });

  // --- Active Filters Computation ---
  let activeFilters = $derived.by(() => {
    const filters = [];
    if (selectedResource !== 'all') {
      filters.push({
        type: 'entity',
        label: resourceFilters.find((f) => f.id === selectedResource)?.label || selectedResource,
      });
    }
    if (selectedAction !== 'all') {
      filters.push({
        type: 'action',
        label: actionFilters.find((f) => f.id === selectedAction)?.label || selectedAction,
      });
    }
    if (selectedDateRange !== '24h') {
      if (selectedDateRange === 'custom') {
        filters.push({ type: 'date', label: `${customStartDate} to ${customEndDate}` });
      } else {
        const dr = dateRanges.find((d) => d.id === selectedDateRange);
        if (dr) filters.push({ type: 'date', label: dr.label });
      }
    }
    return filters;
  });

  function clearFilter(type) {
    if (type === 'entity') selectedResource = 'all';
    if (type === 'action') selectedAction = 'all';
    if (type === 'date') selectedDateRange = '24h';
    offset = 0;
    fetchLogs();
  }

  function clearAllFilters() {
    selectedResource = 'all';
    selectedAction = 'all';
    selectedDateRange = '24h';
    searchQuery = '';
    offset = 0;
    fetchLogs();
  }

  function resetFilters() {
    selectedResource = 'all';
    selectedAction = 'all';
    selectedDateRange = '24h';
    searchQuery = '';
    offset = 0;
    fetchLogs();
  }

  function getFilterIcon(type: string) {
    if (type === 'entity') return Tag;
    if (type === 'action') return List;
    if (type === 'date') return CalendarDays;
    if (type === 'search') return Search;
    return List;
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
    else if (filterChanged || pageChanged) fetchLogs();

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

  // --- Diff computation ---
  function getChanges(log) {
    const ov = log?.old_values || {};
    const nv = log?.new_values || {};
    const keys = new Set([...Object.keys(ov), ...Object.keys(nv)]);
    const out = [];
    for (const key of keys) {
      if (JSON.stringify(ov[key]) === JSON.stringify(nv[key])) continue;
      if (['password', 'password_hash', 'token', 'token_hash'].includes(key.toLowerCase())) continue;
      out.push({ key, old: ov[key], new: nv[key] });
    }
    return out;
  }

  const fieldLabels: Record<string, string> = {
    name: 'Name',
    username: 'Username',
    email: 'Email',
    role: 'Role',
    role_id: 'Role',
    is_active: 'Active Status',
    is_system: 'System Role',
    description: 'Description',
    price: 'Price',
    stock: 'Stock',
    category: 'Category',
    category_id: 'Category',
    barcode: 'Barcode',
    sku: 'SKU',
    quantity_change: 'Quantity Change',
    notes: 'Notes',
    invoice_number: 'Invoice Number',
    status: 'Status',
    payment_method: 'Payment Method',
    discount: 'Discount',
    tax: 'Tax',
    subtotal: 'Subtotal',
    total: 'Total',
    cashier: 'Cashier',
    store: 'Store',
    store_id: 'Store',
    brand: 'Brand',
    brand_id: 'Brand',
    slug: 'Slug',
    parent_id: 'Parent',
    sort_order: 'Sort Order',
    image_url: 'Image URL',
    expiry_date: 'Expiry Date',
    unit: 'Unit',
    weight: 'Weight',
    created_at: 'Created At',
    updated_at: 'Updated At',
    old_password: 'Old Password',
    new_password: 'New Password',
    permission_ids: 'Permissions',
    permission_id: 'Permission',
  };

  function getFieldLabel(key: string) {
    return fieldLabels[key] || key.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
  }

  function formatTimestamp(d) {
    if (!d) return { date: '—', time: '', full: '—' };
    const dateStr = formatDateInJakarta(d);
    const timeStr = formatTimeInJakarta(d);
    return { date: dateStr, time: timeStr, full: `${dateStr} ${timeStr}` };
  }

  function formatDateHuman(d) {
    if (!d) return '—';
    const dateObj = new Date(d);
    const nowMs = Date.now() + JAKARTA_OFFSET_MS;
    const shiftedDate = new Date(dateObj.getTime() + JAKARTA_OFFSET_MS);
    const diffMs = nowMs - shiftedDate.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins} min${diffMins > 1 ? 's' : ''} ago`;
    if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`;
    if (diffDays < 7) return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`;
    return formatDateInJakarta(d);
  }

  function formatValue(val) {
    if (val == null) return '—';
    if (typeof val === 'boolean') return val ? 'Yes' : 'No';
    if (typeof val === 'string') {
      const dateMatch = val.match(/^\d{4}-\d{2}-\d{2}T/);
      if (dateMatch) {
        return formatDateTimeInJakarta(val);
      }
      return val;
    }
    if (typeof val === 'number') {
      if (val > 10000 && Number.isInteger(val)) return 'Rp ' + val.toLocaleString('id-ID');
      return val.toLocaleString('id-ID');
    }
    if (typeof val === 'object') {
      if (Array.isArray(val)) {
        if (val.length === 0) return 'None';
        return val.map((v) => formatValue(v)).join(', ');
      }
      if (val.name) return String(val.name);
      if (val.label) return String(val.label);
      if (val.description) return String(val.description);
      if (val.code) return String(val.code);
      if (val.username) return String(val.username);
      if (val.email) return String(val.email);
      if (val.id != null) {
        const parts = [`ID: ${val.id}`];
        if (val.name) parts.push(val.name);
        else if (val.description) parts.push(val.description);
        else {
          for (const [k, v] of Object.entries(val)) {
            if (k === 'id' || k === 'created_at' || k === 'updated_at' || k === 'is_system') continue;
            if (typeof v !== 'object') {
              parts.push(`${getFieldLabel(k)}: ${formatValue(v)}`);
              if (parts.length >= 3) break;
            }
          }
        }
        return parts.join(' · ');
      }
      const pairs = Object.entries(val)
        .filter(([k]) => k !== 'created_at' && k !== 'updated_at')
        .map(([k, v]) => `${getFieldLabel(k)}: ${formatValue(v)}`);
      return pairs.join(', ') || '—';
    }
    return String(val);
  }

  function getDiffDescription(change) {
    const label = getFieldLabel(change.key);
    const oldVal = formatValue(change.old);
    const newVal = formatValue(change.new);

    if (change.old == null && change.new != null) {
      return { label, text: `Set to "${newVal}"`, icon: Plus, color: 'success' };
    }
    if (change.old != null && change.new == null) {
      return { label, text: `Removed (was "${oldVal}")`, icon: Minus, color: 'danger' };
    }
    return { label, text: `Changed from "${oldVal}" to "${newVal}"`, icon: ArrowRight, color: 'warning' };
  }

  function getActionVerb(action: string) {
    const v = (action || '').toUpperCase();
    if (v === 'CREATE') return 'Created';
    if (v === 'UPDATE') return 'Updated';
    if (v === 'DELETE') return 'Deleted';
    if (v === 'LOGIN') return 'Logged in';
    if (v === 'LOGOUT') return 'Logged out';
    return action;
  }

  function getResourceLabel(entityType: string) {
    const map: Record<string, string> = {
      auth: 'Authentication',
      user: 'User',
      role: 'Role',
      product: 'Product',
      sale: 'Sale',
      category: 'Category',
      brand: 'Brand',
    };
    return map[entityType] || entityType;
  }

  function buildExportUrl(format: string): string {
    const range = getDateRange(selectedDateRange);
    const params = new URLSearchParams({
      format,
      search: searchQuery,
      start_date: range.start.toISOString(),
      end_date: range.end.toISOString(),
    });
    if (selectedAction !== 'all') params.append('action', selectedAction);
    if (selectedResource !== 'all') params.append('entity_type', selectedResource);
    return `/api/audit-logs/export?${params.toString()}`;
  }

  async function downloadExport(format: string) {
    const token = getAuthToken();
    if (!token) { toast.error('Session expired'); return; }

    const res = await fetch(buildExportUrl(format), {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) { toast.error('Export failed'); return; }

    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `audit-logs-${today}.${format === 'csv' ? 'csv' : 'xlsx'}`;
    a.click();
    URL.revokeObjectURL(url);
    showExportDropdown = false;
    toast.success(`Audit logs exported to ${format.toUpperCase()}`);
  }

  function exportToCsv() {
    downloadExport('csv');
  }

  function exportToExcel() {
    downloadExport('xlsx');
  }
</script>

{#if !canView}
  <div class="card px-4 py-16 text-center">
    <ScrollText size={40} class="text-text-muted mx-auto mb-4" />
    <p class="text-text-primary font-semibold text-lg">Access Denied</p>
    <p class="text-text-muted text-sm mt-1">Audit logs are restricted to superadmin only</p>
  </div>
{:else}
  <div class="space-y-5 max-w-7xl mx-auto">
    <!-- Filter Toolbar -->
    <div class="card p-4">
      <div class="flex items-center gap-3">
        <div class="relative flex-1">
          <SearchBar bind:value={searchQuery} placeholder="Search by actor, role, action, entity, or IP..." inputClass="h-10" />
        </div>

        <div class="relative shrink-0 date-picker-container">
          <button
            class="date-picker-trigger btn btn-secondary flex items-center gap-2 min-w-44"
            onclick={() => showDatePicker = !showDatePicker}
          >
            <CalendarDays size={16} class="text-text-secondary shrink-0" />
            <span class="text-sm font-medium truncate flex-1 text-left text-text-secondary">{dateRangeLabel}</span>
            <ChevronDown size={14} class="opacity-60 shrink-0" />
          </button>
          {#if showDatePicker}
            <div class="absolute right-0 top-full mt-1.5 z-50 bg-surface-default border border-border rounded-lg shadow-xl p-3 min-w-64">
              <div class="flex flex-wrap gap-1 mb-3">
                {#each datePresets as preset}
                  <button
                    class="btn btn-ghost btn-xs"
                    onclick={() => applyDatePreset(preset.rangeId)}
                  >
                    {preset.label}
                  </button>
                {/each}
              </div>
              <div class="flex items-center gap-2 text-xs">
                <input type="date" bind:value={customStartDate} class="input input-sm w-full" min={ninetyDaysAgo} max={customEndDate || today} />
                <span class="text-text-muted">—</span>
                <input type="date" bind:value={customEndDate} class="input input-sm w-full" min={customStartDate || ninetyDaysAgo} max={today} />
              </div>
              <div class="flex justify-end mt-2">
                <button
                  class="btn btn-primary btn-xs"
                  onclick={applyCustomDateRange}
                >
                  Apply
                </button>
              </div>
            </div>
          {/if}
        </div>

        <div class="relative shrink-0" style="width: 128px; min-width: 128px; max-width: 128px;" id="resource-dropdown-container">
          <button
            class="flex items-center gap-2 px-3 h-10 w-full rounded-xl border border-border bg-surface-default text-text-secondary text-sm hover:border-border-strong hover:bg-surface-hover transition-colors"
            onclick={() => showResourceDropdown = !showResourceDropdown}
          >
            <span class="flex-1 text-left truncate">{resourceLabel}</span>
            <ChevronDown size={14} class="text-text-muted shrink-0" />
          </button>
          {#if showResourceDropdown}
            <div class="absolute left-0 top-full mt-2 z-50 bg-surface-default border border-border rounded-lg shadow-xl py-1 min-w-[180px]">
              {#each resourceFilters as f}
                <button
                  class="w-full text-left px-4 py-2 text-sm transition-colors {selectedResource === f.id ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
                  onclick={() => { selectedResource = f.id; offset = 0; selectedAction = 'all'; showResourceDropdown = false; }}
                >
                  {f.label}
                </button>
              {/each}
            </div>
          {/if}
        </div>

        <div class="relative shrink-0" style="width: 140px; min-width: 140px; max-width: 140px;" id="action-dropdown-container">
          <button
            class="flex items-center gap-2 px-3 h-10 w-full rounded-xl border border-border bg-surface-default text-text-secondary text-sm hover:border-border-strong hover:bg-surface-hover transition-colors"
            onclick={() => showActionDropdown = !showActionDropdown}
          >
            <span class="flex-1 text-left truncate">{actionLabel}</span>
            <ChevronDown size={14} class="text-text-muted shrink-0" />
          </button>
          {#if showActionDropdown}
            <div class="absolute left-0 top-full mt-2 z-50 bg-surface-default border border-border rounded-lg shadow-xl py-1 min-w-[180px]">
              {#each availableActionFilters as f}
                <button
                  class="w-full text-left px-4 py-2 text-sm transition-colors {selectedAction === f.id ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
                  onclick={() => { selectedAction = f.id; offset = 0; showActionDropdown = false; }}
                >
                  {f.label}
                </button>
              {/each}
            </div>
          {/if}
        </div>

        <button title="Refresh" class="btn btn-secondary px-3 h-10" onclick={fetchLogs}>
          <RefreshCw size={16} class={loading ? 'animate-spin' : ''} />
        </button>
        <!-- Export Dropdown -->
        <div class="relative export-dropdown">
          <button
            class="btn btn-primary flex items-center gap-2 transition-all duration-300 h-10"
            onclick={(e) => {
              e.stopPropagation();
              showExportDropdown = !showExportDropdown;
            }}
            aria-haspopup="menu"
            aria-expanded={showExportDropdown}
          >
            <Download size={15} />
            Export
            <ChevronDown
              size={14}
              class="transition-transform duration-300 {showExportDropdown ? 'rotate-180' : ''}"
            />
          </button>
          {#if showExportDropdown}
            <div
              class="absolute right-0 top-full mt-2 card-glass p-1.5 z-50 min-w-44 flex flex-col gap-0.5 export-dropdown"
              onclick={(e) => e.stopPropagation()}
              onkeydown={(e) => e.stopPropagation()}
              role="menu"
              tabindex="-1"
              transition:fly={{ y: -8, duration: 200 }}
            >
              <button
                class="flex items-center gap-3 px-3 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-xl transition-all duration-200 active:scale-[0.98] w-full text-left"
                role="menuitem"
                onclick={() => {
                  showExportDropdown = false;
                  exportToCsv();
                }}
              >
                <FileSpreadsheet size={16} class="text-success-light" />
                Export to CSV
              </button>
              <button
                class="flex items-center gap-3 px-3 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-xl transition-all duration-200 active:scale-[0.98] w-full text-left"
                role="menuitem"
                onclick={() => {
                  showExportDropdown = false;
                  exportToExcel();
                }}
              >
                <FileSpreadsheet size={16} class="text-info-light" />
                Export to Excel
              </button>
            </div>
          {/if}
        </div>
      </div>

      <div class="filter-chips-wrapper" class:is-open={activeFilters.length > 0}>
        <div class="filter-chips-inner">
          <div class="flex flex-wrap items-center gap-2 pt-3 mt-3 border-t border-border/50">
            {#each activeFilters as filter}
              {@const FilterIcon = getFilterIcon(filter.type)}
              <div class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-primary-subtle/20 border border-primary-subtle/30 rounded-full text-sm text-text-secondary">
                <FilterIcon size={13} class="text-primary-light shrink-0" />
                <span class="font-medium truncate max-w-[180px]">{filter.label}</span>
                <button
                  class="w-4 h-4 rounded-full flex items-center justify-center text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
                  title={`Clear ${filter.label}`}
                  onclick={() => clearFilter(filter.type)}
                  aria-label={`Clear ${filter.label} filter`}
                >
                  <X size={12} />
                </button>
              </div>
            {/each}
            <button
              class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-text-muted hover:text-text-primary bg-surface-default/50 border border-border/50 rounded-full transition-colors"
              onclick={clearAllFilters}
            >
              Clear all
              <X size={12} />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 3. Audit Log Table -->
    <div>
      {#if loading}
        <div class="card p-0 overflow-hidden">
          <div class="divide-y divide-border/70">
            {#each { length: 8 } as _}
              <div class="flex items-center h-9 px-4">
                <div class="w-[180px] shrink-0">
                  <Skeleton width="w-32" height="h-3" />
                </div>
                <div class="w-[180px] shrink-0 flex items-center gap-2">
                  <Skeleton width="w-6" height="h-6" rounded="rounded-full" />
                  <Skeleton width="w-24" height="h-3" />
                </div>
                <div class="w-[120px] shrink-0">
                  <Skeleton width="w-20" height="h-3" />
                </div>
                <div class="w-[120px] shrink-0">
                  <Skeleton width="w-16" height="h-3.5" rounded="rounded-full" />
                </div>
                <div class="flex-1">
                  <Skeleton width="w-full max-w-sm" height="h-3" />
                </div>
                <div class="w-[150px] shrink-0">
                  <Skeleton width="w-24" height="h-3" />
                </div>
              </div>
            {/each}
          </div>
        </div>
      {:else if items.length === 0}
        <div class="card px-4 py-24 flex flex-col items-center justify-center text-center">
          <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
            <Search size={32} class="text-text-muted" />
          </div>
          <p class="text-text-primary font-semibold mt-4">No audit logs found</p>
          <p class="text-text-muted text-sm mt-1 max-w-sm">
            Try adjusting your filters or search terms to find what you're looking for.
          </p>
          <button class="btn btn-secondary mt-6" onclick={clearAllFilters}> Clear Filters </button>
        </div>
      {:else}
        <div class="card p-0 overflow-hidden">
          <div class="overflow-x-auto">
            <table class="w-full text-sm text-left whitespace-nowrap">
              <thead class="sticky top-0 bg-bg-secondary z-10 shadow-sm">
                <tr class="text-[11px] font-semibold text-text-muted uppercase tracking-wider h-10">
                  <th class="px-4 py-3 w-[180px]">Timestamp</th>
                  <th class="px-4 py-3 w-[180px]">Actor</th>
                  <th class="px-4 py-3 w-[120px]">Resource</th>
                  <th class="px-4 py-3 w-[120px]">Action</th>
                  <th class="px-4 py-3">Description</th>
                  <th class="px-4 py-3 w-[150px]">IP Address</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border/70">
                {#each items as log (log.id)}
                  {@const ts = formatTimestamp(log.created_at)}
                  <tr
                    class="h-10 px-4 leading-none border-t border-border/70 hover:bg-surface-hover/50 transition-colors cursor-pointer"
                    onclick={() => openDrawer(log)}
                  >
                     <td class="py-2 align-middle">
                       <span class="text-text-primary font-medium text-sm leading-snug">{ts.date}</span>
                       <span class="block text-text-muted text-[10px]">{ts.time}</span>
                     </td>
                     <td class="py-2 align-middle">
                       {#if log.username && log.username !== '—'}
                         <div class="flex items-center gap-2">
                           <div class="w-6 h-6 rounded-full gradient-bg-primary flex items-center justify-center shrink-0">
                             <span class="text-[10px] font-bold text-white">{log.username.charAt(0).toUpperCase()}</span>
                           </div>
                           <span class="font-medium text-text-primary text-sm truncate max-w-[130px]">{log.username}</span>
                         </div>
                       {:else}
                         <span class="font-medium text-text-muted text-sm">—</span>
                       {/if}
                     </td>
                     <td class="py-2 align-middle">
                       <span
                         class="font-mono text-sm text-text-secondary bg-surface-hover px-2 py-1 rounded border border-border/50 capitalize"
                       >
                         {log.entity_type || '—'}
                       </span>
                     </td>
                     <td class="py-2 align-middle">
                       <ActionBadge action={log.action} />
                     </td>
                     <td class="py-2 align-middle text-sm text-text-secondary truncate max-w-xs leading-snug" title={log.description}>
                       {log.description || '—'}
                     </td>
                     <td class="py-2 align-middle font-mono text-[10px] text-text-muted leading-none">
                       {log.ip_address || '—'}
                     </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>

          <div class="p-4 bg-surface-subtle/30 border-t border-border/50">
            <Pagination {total} {limit} {offset} onPageChange={handlePageChange} />
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- Details Drawer -->
{#if drawerOpen && selectedLog}
  {@const changes = getChanges(selectedLog)}
  <button type="button" class="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm" onclick={closeDrawer} aria-label="Close drawer"></button>
  <div class="fixed right-0 top-0 bottom-0 w-full max-w-lg z-50 bg-bg border-l border-border shadow-2xl flex flex-col animate-slide-in">
    <!-- Header -->
    <div class="flex items-center justify-between px-5 py-4 border-b border-border shrink-0">
      <div class="flex items-center gap-3">
        <ActionBadge action={selectedLog.action} />
        <span class="font-mono text-sm text-text-muted bg-surface-default px-2 py-0.5 rounded border border-border/50">{selectedLog.entity_type}</span>
      </div>
      <button class="p-1.5 rounded-lg text-text-muted hover:text-text-secondary hover:bg-surface-hover transition-colors" onclick={closeDrawer}>
        <X size={18} />
      </button>
    </div>

    <!-- Body -->
    <div class="flex-1 overflow-y-auto px-5 py-4 space-y-5">
      <!-- Human-friendly summary -->
      <div class="bg-surface-default rounded-lg p-4 border border-border/50">
        <p class="text-sm text-text-primary leading-relaxed">
          <span class="font-semibold">{selectedLog.username || 'Unknown user'}</span>
          {#if selectedLog.role}<span class="text-text-muted"> ({selectedLog.role})</span>{/if}
          <span> </span>
          <span class="font-medium">{getActionVerb(selectedLog.action)}</span>
          {#if selectedLog.entity_type}
            <span> a </span>
            <span class="font-medium">{getResourceLabel(selectedLog.entity_type)}</span>
          {/if}
          {#if selectedLog.entity_id}
            <span> (ID: {selectedLog.entity_id})</span>
          {/if}
        </p>
        <p class="text-xs text-text-muted mt-2 flex items-center gap-1.5">
          <Clock size={12} />
          {formatDateHuman(selectedLog.created_at)} · {formatTimestamp(selectedLog.created_at).full}
        </p>
      </div>

      <!-- Description -->
      {#if selectedLog.description}
        <div>
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">Description</p>
          <p class="text-sm text-text-primary">{selectedLog.description}</p>
        </div>
      {/if}

      <!-- Meta grid -->
      <div class="grid grid-cols-2 gap-3">
        <div class="bg-surface-default rounded-lg p-3 border border-border/50">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">When</p>
          <p class="text-sm text-text-primary">{formatTimestamp(selectedLog.created_at).full}</p>
          <p class="text-xs text-text-muted mt-0.5">{formatDateHuman(selectedLog.created_at)}</p>
        </div>
        <div class="bg-surface-default rounded-lg p-3 border border-border/50">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">Who</p>
          <div class="flex items-center gap-2">
            {#if selectedLog.username && selectedLog.username !== '—'}
              <div class="w-5 h-5 rounded-full gradient-bg-primary flex items-center justify-center shrink-0">
                <span class="text-[8px] font-bold text-white">{selectedLog.username.charAt(0).toUpperCase()}</span>
              </div>
            {/if}
            <p class="text-sm text-text-primary">{selectedLog.username || 'Unknown'}</p>
          </div>
          {#if selectedLog.role}
            <p class="text-xs text-text-secondary mt-0.5 capitalize">{selectedLog.role}</p>
          {/if}
        </div>
        <div class="bg-surface-default rounded-lg p-3 border border-border/50">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">From</p>
          <div class="flex items-center gap-1.5 text-sm text-text-primary">
            <Globe size={14} class="text-text-muted" />
            <span class="font-mono">{selectedLog.ip_address || '—'}</span>
          </div>
        </div>
        <div class="bg-surface-default rounded-lg p-3 border border-border/50">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">Resource</p>
          <p class="text-sm text-text-primary capitalize">{getResourceLabel(selectedLog.entity_type) || '—'}</p>
          {#if selectedLog.entity_id}
            <p class="text-xs text-text-secondary font-mono mt-0.5">ID: {selectedLog.entity_id}</p>
          {/if}
        </div>
      </div>

      <!-- User Agent -->
      {#if selectedLog.user_agent}
        <div>
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-2">Browser / Device</p>
          <div class="flex items-start gap-2 p-3 bg-surface-default rounded-lg border border-border/50">
            <Monitor size={14} class="text-text-muted mt-0.5 shrink-0" />
            <p class="text-xs text-text-secondary font-mono leading-relaxed break-all">{selectedLog.user_agent}</p>
          </div>
        </div>
      {/if}

      <!-- Data Changes -->
      {#if changes.length > 0}
        <div>
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-3">What Changed</p>
          <div class="space-y-2">
            {#each changes as change}
              {@const diff = getDiffDescription(change)}
              <div class="bg-surface-default rounded-lg p-3 border border-border/50">
                <div class="flex items-start gap-3">
                  <div class="w-6 h-6 rounded-full flex items-center justify-center shrink-0 mt-0.5 {diff.color === 'success' ? 'bg-success-subtle' : diff.color === 'danger' ? 'bg-danger-subtle' : 'bg-warning-subtle'}">
                    <diff.icon
                      size={12}
                      class={diff.color === 'success'
                        ? 'text-success-light'
                        : diff.color === 'danger'
                          ? 'text-danger-light'
                          : 'text-warning-light'}
                    />
                  </div>
                  <div class="flex-1 min-w-0">
                    <p class="text-xs font-semibold text-text-secondary">{diff.label}</p>
                    <p class="text-sm text-text-primary mt-0.5">{diff.text}</p>
                  </div>
                </div>
              </div>
            {/each}
          </div>
        </div>
      {:else if selectedLog.action === 'CREATE' || selectedLog.action === 'UPDATE' || selectedLog.action === 'DELETE'}
        <div class="p-4 text-center bg-surface-default/50 rounded-lg border border-dashed border-border/40">
          <p class="text-sm text-text-muted">No specific data changes captured for this {selectedLog.action.toLowerCase()} action.</p>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  @keyframes slide-in {
    from {
      transform: translateX(100%);
    }
    to {
      transform: translateX(0);
    }
  }
  .animate-slide-in {
    animation: slide-in 0.2s ease-out;
  }

  :global(input[type="date"]::-webkit-calendar-picker-indicator) {
    filter: invert(1);
    cursor: pointer;
  }

  .filter-chips-wrapper {
    display: grid;
    grid-template-rows: 0fr;
    opacity: 0;
    transition: grid-template-rows 0.2s ease-out, opacity 0.2s ease-out;
  }

  .filter-chips-wrapper.is-open {
    grid-template-rows: 1fr;
    opacity: 1;
  }

  .filter-chips-inner {
    overflow: hidden;
  }
</style>
