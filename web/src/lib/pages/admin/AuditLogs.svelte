<script>
  import { onMount } from 'svelte';
  import apiClient from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { debounce } from '$lib/utils/debounce';
  import { auth } from '$lib/stores/auth';

  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import {
    Search, ScrollText, RefreshCw, CalendarDays, X, List,
    Plus, Edit, Trash, LogIn, LogOut, Info, ArrowRight,
    Monitor, Globe, ChevronRight, Package, ShoppingCart,
    Users, Tag, Shield, Store, Calendar, ChevronDown, ExternalLink,
  } from 'lucide-svelte';

  let loading = $state(true);
  let items = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let selectedAction = $state('all');
  let selectedEntity = $state('all');
  let selectedDateRange = $state('7d');
  let showDateDropdown = $state(false);
  let showDetailsDrawer = $state(false);
  let selectedLog = $state(null);

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
  let isDrawerOpen = $state(false);

  // --- Filter definitions ---

  const actionFilters = [
    { id: 'all', label: 'All', icon: List },
    { id: 'create', label: 'Create', icon: Plus },
    { id: 'update', label: 'Update', icon: Edit },
    { id: 'delete', label: 'Delete', icon: Trash },
    { id: 'login', label: 'Login', icon: LogIn },
    { id: 'logout', label: 'Logout', icon: LogOut },
  ];

  const entityFilters = [
    { id: 'all', label: 'All', icon: List },
    { id: 'auth', label: 'Auth', icon: LogIn },
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

  let customStartDate = $state('');
  let customEndDate = $state('');
  let showCustomDateModal = $state(false);

  const today = new Date().toISOString().split('T')[0];
  const ninetyDaysAgo = new Date(Date.now() - 90 * 86400000).toISOString().split('T')[0];

  let startDateMin = $derived(ninetyDaysAgo);
  let startDateMax = $derived(customEndDate || today);
  let endDateMin = $derived(customStartDate || ninetyDaysAgo);
  let endDateMax = $derived(today);

  function getDateRange(range) {
    const now = new Date();
    switch (range) {
      case '24h': return { start: new Date(now - 86400000), end: now };
      case '7d': return { start: new Date(now - 7 * 86400000), end: now };
      case '30d': return { start: new Date(now - 30 * 86400000), end: now };
      case '90d': return { start: new Date(now - 90 * 86400000), end: now };
      case 'custom':
        if (customStartDate && customEndDate) {
          return { start: new Date(customStartDate), end: new Date(customEndDate + 'T23:59:59') };
        }
        return { start: new Date(now - 7 * 86400000), end: now };
      default: return { start: new Date(now - 7 * 86400000), end: now };
    }
  }

  function applyCustomRange() {
    if (customStartDate && customEndDate) {
      showCustomDateModal = false;
      showDateDropdown = false;
      selectedDateRange = 'custom';
      offset = 0;
      fetchLogs();
    }
  }

  function selectDateRange(rangeId) {
    if (rangeId === 'custom') {
      showDateDropdown = false;
      showCustomDateModal = true;
      const now = new Date();
      const weekAgo = new Date(now - 7 * 86400000);
      customEndDate = now.toISOString().split('T')[0];
      customStartDate = weekAgo.toISOString().split('T')[0];
    } else {
      selectedDateRange = rangeId;
      showDateDropdown = false;
      offset = 0;
      fetchLogs();
    }
  }

  $effect(() => {
    if (!showDateDropdown) return;

    const container = document.getElementById('date-dropdown-container');
    const handleClickOutside = (e) => {
      if (container && !container.contains(e.target)) {
        showDateDropdown = false;
      }
    };
    const handleEsc = (e) => {
      if (e.key === 'Escape') showDateDropdown = false;
    };

    document.addEventListener('click', handleClickOutside);
    document.addEventListener('keydown', handleEsc);

    return () => {
      document.removeEventListener('click', handleClickOutside);
      document.removeEventListener('keydown', handleEsc);
    };
  });

  // --- Badge variant for action column ---
  const actionVariant = (a) => {
    if (!a) return 'muted';
    const v = a.toUpperCase();
    if (v === 'CREATE') return 'success';
    if (v === 'DELETE') return 'danger';
    if (v === 'UPDATE') return 'warning';
    if (v === 'LOGIN' || v === 'LOGOUT') return 'primary';
    return 'muted';
  };

  const actionVariantClass = (a) => {
    const v = (a || '').toUpperCase();
    if (v === 'CREATE') return 'bg-success-subtle text-success-light border border-success-default/20';
    if (v === 'DELETE') return 'bg-danger-subtle text-danger-light border border-danger-default/20';
    if (v === 'UPDATE') return 'bg-warning-subtle text-warning-light border border-warning-default/20';
    if (v === 'LOGIN' || v === 'LOGOUT') return 'bg-primary-subtle text-primary-light border border-primary-default/20';
    return 'bg-surface-default text-text-muted border border-border-default';
  };

  const actionIcon = (a) => {
    const v = (a || '').toLowerCase();
    if (v === 'create') return Plus;
    if (v === 'update') return Edit;
    if (v === 'delete') return Trash;
    if (v === 'login') return LogIn;
    if (v === 'logout') return LogOut;
    return Info;
  };

  const entityIcon = (entity) => {
    const v = (entity || '').toLowerCase();
    if (v === 'auth') return Shield;
    if (v === 'user') return Users;
    if (v === 'role') return Shield;
    if (v === 'product') return Package;
    if (v === 'sale') return ShoppingCart;
    if (v === 'category') return Tag;
    if (v === 'brand') return Store;
    return ScrollText;
  };

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
      if (selectedEntity !== 'all') params.append('entity_type', selectedEntity);

      console.debug('[AuditLogs] fetching', `audit-logs?${params.toString()}`, 'role=', userRole, 'canView=', canView);

      const response = await apiClient.get(`audit-logs?${params.toString()}`);

      if (requestId !== currentRequestId) return;

      console.debug('[AuditLogs] response', response.status, response.data);
      const data = response.data || {};
      items = data.data || [];
      total = data.total || 0;
      console.debug('[AuditLogs] items', items.length, 'total', total);
    } catch (error) {
      console.error('[AuditLogs] fetch error', error);
      if (error.name !== 'AbortError') toast.error('Failed to load audit logs');
    } finally {
      if (requestId === currentRequestId) {
        loading = false;
        abortController = null;
      }
    }
  }

  // --- Effect: watch all filter changes ---
  let prevSearch = $state('');
  let prevAction = $state('all');
  let prevEntity = $state('all');
  let prevDate = $state('7d');
  let prevOff = $state(0);
  let prevLim = $state(20);

  const debouncedSearchFetch = debounce(() => { offset = 0; fetchLogs(); }, 400);

  $effect(() => {
    const sq = searchQuery, sa = selectedAction, se = selectedEntity, sd = selectedDateRange, so = offset, sl = limit;

    if (!hasInitialized) {
      hasInitialized = true;
      fetchLogs();
      prevSearch = sq; prevAction = sa; prevEntity = se; prevDate = sd; prevOff = so; prevLim = sl;
      return;
    }

    const searchChanged = sq !== prevSearch;
    const filterChanged = sa !== prevAction || se !== prevEntity || sd !== prevDate;
    const pageChanged = so !== prevOff || sl !== prevLim;

    if (searchChanged) debouncedSearchFetch();
    else if (filterChanged || pageChanged) fetchLogs();

    prevSearch = sq; prevAction = sa; prevEntity = se; prevDate = sd; prevOff = so; prevLim = sl;
  });

  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
  }

  function openDetails(log) {
    selectedLog = log;
    isDrawerOpen = true;
  }

  function closeDetails() {
    isDrawerOpen = false;
    selectedLog = null;
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

  function formatTimestamp(d) {
    if (!d) return { date: '—', time: '' };
    const dateObj = new Date(d);
    const dateStr = dateObj.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
    const timeStr = dateObj.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
    return { date: dateStr, time: timeStr };
  }
  let changes = $derived(selectedLog ? getChanges(selectedLog) : []);
</script>

{#if !canView}
  <div class="card px-4 py-16 text-center">
    <ScrollText size={40} class="text-text-muted mx-auto mb-4" />
    <p class="text-text-primary font-semibold text-lg">Access Denied</p>
    <p class="text-text-muted text-sm mt-1">Audit logs are restricted to superadmin only</p>
  </div>
{:else}
  <div class="space-y-4">
    <!-- ─── Filter Toolbar ─── -->
    <div class="card p-3">
      <!-- Row 1: Search + Date + Refresh -->
      <div class="flex items-center gap-3 mb-3">
        <div class="relative flex-1">
          <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
          <input
            type="text"
            placeholder="Search user, resource, description, IP..."
            class="input w-full pl-9 pr-9 py-2 text-sm"
            bind:value={searchQuery}
          />
          {#if searchQuery}
            <button
              onclick={() => searchQuery = ''}
              class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
            >
              <X size={14} />
            </button>
          {/if}
        </div>

        <!-- Date range dropdown -->
        <div class="relative" id="date-dropdown-container">
          <button
            class="flex items-center gap-2 px-3 py-2 rounded-lg border border-border bg-surface-default text-text-secondary text-sm hover:border-border-strong transition-colors"
            onclick={() => showDateDropdown = !showDateDropdown}
          >
            <Calendar size={14} />
            <span>{dateRanges.find(d => d.id === selectedDateRange)?.label || 'Last 7 Days'}</span>
            <ChevronDown size={12} />
          </button>
          {#if showDateDropdown}
            <div class="absolute right-0 top-full mt-1 z-50 bg-surface-default border border-border rounded-lg shadow-lg py-1 min-w-[180px]">
              {#each dateRanges as range}
                <button
                  class="w-full text-left px-3 py-2 text-sm hover:bg-surface-hover transition-colors {selectedDateRange === range.id ? 'text-primary-light bg-primary-subtle/20' : 'text-text-secondary'}"
                  onclick={() => selectDateRange(range.id)}
                >
                  {range.label}
                </button>
              {/each}
            </div>
          {/if}
        </div>

        <!-- Refresh (secondary) -->
        <button
          title="Refresh"
          class="p-2 rounded-lg text-text-muted hover:text-text-secondary hover:bg-surface-hover border border-transparent hover:border-border transition-all"
          onclick={fetchLogs}
        >
          <RefreshCw size={16} class="{loading ? 'animate-spin' : ''}" />
        </button>
      </div>

      <!-- Row 2: Action + Resource filter chips -->
      <div class="flex flex-col gap-2">
        <!-- Action -->
        <div class="flex items-center gap-2">
          <span class="text-[11px] font-semibold text-text-muted uppercase tracking-wider w-14 shrink-0">Action</span>
          <div class="flex flex-wrap gap-1.5">
            {#each actionFilters as f}
              {@const Icon = f.icon}
              <button
                class="flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium transition-all {selectedAction === f.id ? 'bg-primary-subtle text-primary-light border border-primary/30' : 'bg-surface-default text-text-muted border border-border hover:border-border-strong hover:text-text-secondary'}"
                onclick={() => { selectedAction = f.id; offset = 0; }}
              >
                <Icon size={12} />
                {f.label}
              </button>
            {/each}
          </div>
        </div>

        <!-- Resource -->
        <div class="flex items-center gap-2">
          <span class="text-[11px] font-semibold text-text-muted uppercase tracking-wider w-14 shrink-0">Resource</span>
          <div class="flex flex-wrap gap-1.5">
            {#each entityFilters as f}
              {@const Icon = f.icon}
              <button
                class="flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium transition-all {selectedEntity === f.id ? 'bg-primary-subtle text-primary-light border border-primary/30' : 'bg-surface-default text-text-muted border border-border hover:border-border-strong hover:text-text-secondary'}"
                onclick={() => { selectedEntity = f.id; offset = 0; }}
              >
                <Icon size={12} />
                {f.label}
              </button>
            {/each}
          </div>
        </div>
      </div>
    </div>

    <!-- ─── Table ─── -->
    <div class="card p-0 overflow-hidden">
      <div class="px-4 py-2.5 border-b border-border flex items-center justify-between">
        <p class="text-sm font-medium text-text-primary">Activity Log</p>
        {#if !loading && total > 0}
          <span class="text-xs text-text-muted">{total} entries</span>
        {/if}
      </div>

      {#if loading}
        <div class="divide-y divide-border">
          {#each { length: 6 } as _}
            <div class="flex items-center gap-3 px-4 py-3">
              <div class="space-y-1.5 w-36">
                <Skeleton width="w-full" height="h-3" />
                <Skeleton width="w-20" height="h-2.5" />
              </div>
              <div class="flex items-center gap-2">
                <Skeleton width="w-7" height="h-7" rounded="rounded-full" />
                <Skeleton width="w-16" height="h-3" />
              </div>
              <Skeleton width="w-16" height="h-5" rounded="rounded-full" />
              <div class="flex items-center gap-1.5">
                <Skeleton width="w-4" height="h-4" rounded="rounded" />
                <Skeleton width="w-20" height="h-3.5" />
              </div>
              <Skeleton width="w-40" height="h-3" />
              <Skeleton width="w-24" height="h-3" />
              <Skeleton width="w-7" height="h-7" rounded="rounded-lg" />
            </div>
          {/each}
        </div>
      {:else if items.length === 0}
        <div class="px-4 py-16 text-center">
          <div class="empty-state-icon bg-surface w-20 h-20 mx-auto">
            <ScrollText size={32} class="text-text-muted" />
          </div>
          <p class="text-text-primary font-semibold mt-4">No log entries found</p>
          <p class="text-text-muted text-sm mt-1">
            {searchQuery ? `No results for "${searchQuery}"` : 'Try adjusting your filters'}
          </p>
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead class="sticky top-0 bg-bg-secondary z-10">
              <tr>
                <th class="text-left px-4 py-3 text-[11px] font-semibold text-text-muted uppercase tracking-wider w-44">Timestamp</th>
                <th class="text-left px-3 py-3 text-[11px] font-semibold text-text-muted uppercase tracking-wider w-28">Actor</th>
                <th class="text-left px-3 py-3 text-[11px] font-semibold text-text-muted uppercase tracking-wider w-20">Action</th>
                <th class="text-left px-3 py-3 text-[11px] font-semibold text-text-muted uppercase tracking-wider w-28">Resource</th>
                <th class="text-left px-3 py-3 text-[11px] font-semibold text-text-muted uppercase tracking-wider">Description</th>
                <th class="text-left px-3 py-3 text-[11px] font-semibold text-text-muted uppercase tracking-wider w-32">IP Address</th>
                <th class="w-10 px-2 py-3"></th>
              </tr>
            </thead>
            <tbody>
              {#each items as log (log.id)}
                {@const EntityIcon = entityIcon(log.entity_type)}
                {@const ActionIcon = actionIcon(log.action)}
                <tr class="border-t border-border/60 hover:bg-surface-hover/50 transition-colors group">
                  <td class="px-4 py-3 text-xs whitespace-nowrap">
                    {#if log.created_at}
                      {@const ts = formatTimestamp(log.created_at)}
                      <div class="flex flex-col">
                        <span class="text-text-secondary font-medium">{ts.date}</span>
                        <span class="text-text-muted text-[11px]">{ts.time}</span>
                      </div>
                    {:else}
                      <span class="text-text-muted">—</span>
                    {/if}
                  </td>
                  <td class="px-3 py-3">
                    {#if log.username && log.username !== '—'}
                      <div class="flex items-center gap-2.5">
                        <div class="w-7 h-7 rounded-full gradient-bg-primary flex items-center justify-center shrink-0">
                          <span class="text-xs font-bold text-white">{log.username.charAt(0).toUpperCase()}</span>
                        </div>
                        <span class="font-medium text-text-primary text-xs">{log.username}</span>
                      </div>
                    {:else}
                      <span class="font-medium text-text-muted text-xs">—</span>
                    {/if}
                  </td>
                  <td class="px-4 py-3">
                    <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium {actionVariantClass(log.action)}">
                      <ActionIcon size={11} />
                      {log.action || '—'}
                    </span>
                  </td>
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-1.5">
                      <EntityIcon size={12} class="text-text-muted shrink-0" />
                      <span class="font-mono text-xs text-text-muted bg-surface-default px-1.5 py-0.5 rounded border border-border/50 truncate max-w-[100px]">
                        {log.entity_type || '—'}
                      </span>
                    </div>
                  </td>
                  <td class="px-4 py-3 text-xs text-text-secondary" style="max-width:200px;display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden">
                    {log.description || '—'}
                  </td>
                  <td class="px-4 py-3 font-mono text-xs text-text-muted">
                    {log.ip_address || '—'}
                  </td>
                  <td class="px-2 py-3">
                    <button
                      class="p-1.5 rounded-md text-text-muted/70 hover:text-primary-light hover:bg-primary-subtle/20 transition-all"
                      onclick={() => openDetails(log)}
                      title="View Details"
                    >
                      <ExternalLink size={13} />
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <div class="px-4 py-3 bg-surface-subtle/20 border-t border-border/60">
          <Pagination {total} {limit} {offset} onPageChange={handlePageChange} />
        </div>
      {/if}
    </div>
  </div>

  <!-- ─── Details Drawer ─── -->
  {#if isDrawerOpen && selectedLog}
    <!-- Backdrop -->
    <div class="fixed inset-0 z-40 bg-black/30 backdrop-blur-sm" onclick={closeDetails}></div>

    <!-- Drawer panel -->
    <div class="fixed right-0 top-0 bottom-0 w-full max-w-lg z-50 bg-bg border-l border-border shadow-2xl flex flex-col animate-slide-in">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-border">
        <div class="flex items-center gap-3">
          <Badge variant={actionVariant(selectedLog.action)} size="sm">{selectedLog.action}</Badge>
          <span class="font-mono text-xs text-text-muted bg-surface-default px-2 py-0.5 rounded border border-border/50">{selectedLog.entity_type}</span>
        </div>
        <button class="p-1.5 rounded-lg text-text-muted hover:text-text-secondary hover:bg-surface-hover transition-colors" onclick={closeDetails}>
          <X size={18} />
        </button>
      </div>

      <!-- Body -->
      <div class="flex-1 overflow-y-auto px-5 py-4 space-y-5">
        <!-- Description -->
        <div>
          <p class="text-text-primary font-medium">{selectedLog.description}</p>
        </div>

        {#if selectedLog.entity_id}
          <div class="text-xs text-text-muted font-mono">Entity ID: {selectedLog.entity_id}</div>
        {/if}

        <!-- Meta grid -->
        <div class="grid grid-cols-2 gap-3">
          <div class="bg-surface-default rounded-lg p-3 border border-border/50">
            <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">Timestamp</p>
            <div class="flex flex-col">
              {#if selectedLog.created_at}
                {@const ts = formatTimestamp(selectedLog.created_at)}
                <span class="font-medium">{ts.date}</span>
                <span class="text-text-muted text-[11px]">{ts.time}</span>
              {:else}
                <span>—</span>
              {/if}
            </div>
          </div>
          <div class="bg-surface-default rounded-lg p-3 border border-border/50">
            <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">Actor</p>
            <p class="text-xs text-text-primary font-medium">{selectedLog.username || '—'}</p>
          </div>
          <div class="bg-surface-default rounded-lg p-3 border border-border/50">
            <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">IP Address</p>
            <div class="flex items-center gap-1.5 text-xs text-text-secondary font-mono">
              <Globe size={12} />
              {selectedLog.ip_address || '—'}
            </div>
          </div>
          <div class="bg-surface-default rounded-lg p-3 border border-border/50">
            <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">Role</p>
            <p class="text-xs text-text-secondary">{selectedLog.role || '—'}</p>
          </div>
        </div>

        <!-- User Agent -->
        {#if selectedLog.user_agent}
          <div>
            <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-2">User Agent</p>
            <div class="flex items-start gap-2 p-3 bg-surface-default rounded-lg border border-border/50">
              <Monitor size={14} class="text-text-muted mt-0.5 shrink-0" />
              <p class="text-xs text-text-secondary font-mono leading-relaxed break-all">{selectedLog.user_agent}</p>
            </div>
          </div>
        {/if}

        <!-- Data Changes -->
        <div>
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-2">Data Changes</p>
          {#if changes.length > 0}
            <div class="space-y-2">
              {#each changes as change}
                <div class="border border-border/50 rounded-lg overflow-hidden">
                  <div class="bg-surface-default px-3 py-1.5 border-b border-border/50">
                    <span class="text-[11px] font-semibold text-text-secondary">{change.key.replace(/_/g, ' ')}</span>
                  </div>
                  <div class="grid grid-cols-2 divide-x divide-border/50">
                    <div class="p-2.5">
                      <p class="text-[9px] uppercase tracking-widest text-text-muted font-semibold mb-1">Before</p>
                      <p class="text-[11px] font-mono text-danger-light break-all">
                        {change.old != null ? JSON.stringify(change.old) : '—'}
                      </p>
                    </div>
                    <div class="p-2.5">
                      <p class="text-[9px] uppercase tracking-widest text-text-muted font-semibold mb-1">After</p>
                      <p class="text-[11px] font-mono text-success-light break-all">
                        {change.new != null ? JSON.stringify(change.new) : '—'}
                      </p>
                    </div>
                  </div>
                </div>
              {/each}
            </div>
          {:else if selectedLog.action === 'CREATE' || selectedLog.action === 'UPDATE' || selectedLog.action === 'DELETE'}
            <div class="p-6 text-center bg-surface-default/50 rounded-lg border border-dashed border-border/40">
              <p class="text-xs text-text-muted italic">No specific data changes captured.</p>
            </div>
          {:else}
            <div class="p-6 text-center bg-surface-default/50 rounded-lg border border-dashed border-border/40">
              <p class="text-xs text-text-muted italic">System event — no data changes.</p>
            </div>
          {/if}
        </div>
      </div>
    </div>
  {/if}
{/if}

<!-- Custom Date Range Modal -->
{#if showCustomDateModal}
<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" onclick={() => showCustomDateModal = false}>
  <div class="bg-surface-default border border-border rounded-xl shadow-2xl p-5 w-full max-w-sm mx-4" onclick={(e) => e.stopPropagation()}>
    <h3 class="text-sm font-semibold text-text-primary mb-4">Custom Date Range</h3>
    <div class="space-y-3">
      <div>
        <label for="custom-start" class="block text-xs font-medium text-text-secondary mb-1">Start Date</label>
        <input
          id="custom-start"
          type="date"
          class="input w-full text-sm"
          bind:value={customStartDate}
          min={startDateMin}
          max={startDateMax}
        />
      </div>
      <div>
        <label for="custom-end" class="block text-xs font-medium text-text-secondary mb-1">End Date</label>
        <input
          id="custom-end"
          type="date"
          class="input w-full text-sm"
          bind:value={customEndDate}
          min={endDateMin}
          max={endDateMax}
        />
      </div>
    </div>
    <div class="flex items-center justify-end gap-2 mt-5">
      <button class="btn btn-secondary text-sm px-4 py-2" onclick={() => showCustomDateModal = false}>Cancel</button>
      <button class="btn btn-primary text-sm px-4 py-2" onclick={applyCustomRange} disabled={!customStartDate || !customEndDate}>Apply</button>
    </div>
  </div>
</div>
{/if}

<style>
  @keyframes slide-in {
    from { transform: translateX(100%); }
    to { transform: translateX(0); }
  }
  .animate-slide-in {
    animation: slide-in 0.2s ease-out;
  }

  #custom-start::-webkit-calendar-picker-indicator,
  #custom-end::-webkit-calendar-picker-indicator {
    filter: invert(1);
  }
</style>
