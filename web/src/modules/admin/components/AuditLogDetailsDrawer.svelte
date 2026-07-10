<script lang="ts">
  import { X, Plus, Minus, ArrowRight, Clock, Globe, Monitor } from 'lucide-svelte';
  import { ActionBadge } from '$shared/ui';
  import { formatDateInJakarta, formatTimeInJakarta, formatDateTimeInJakarta, JAKARTA_OFFSET_MS } from '$shared/utils/jakartaTime';
  import { fly } from 'svelte/transition';

  let {
    selectedLog = null,
    drawerOpen = $bindable(false),
    onclose = () => {},
  }: {
    selectedLog?: any;
    drawerOpen?: boolean;
    onclose?: () => void;
  } = $props();

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
      stock: 'Stock',
      uom: 'Unit of Measure',
      customer: 'Customer',
    };
    return map[entityType] || entityType;
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
</script>

{#if drawerOpen && selectedLog}
  {@const changes = getChanges(selectedLog)}
  <button type="button" class="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm" onclick={onclose} aria-label="Close drawer"></button>
  <div class="fixed right-0 top-0 bottom-0 w-full max-w-lg z-50 bg-bg border-l border-border shadow-2xl flex flex-col animate-slide-in" role="dialog" aria-modal="true" aria-label="Audit Log Details">
    <!-- Header -->
    <div class="flex items-center justify-between px-5 py-4 border-b border-border shrink-0">
      <div class="flex items-center gap-3">
        <ActionBadge action={selectedLog.action} />
        <span class="font-mono text-sm text-text-muted bg-surface-default px-2 py-0.5 rounded border border-border/50">{selectedLog.entity_type}</span>
      </div>
      <button type="button" class="p-1.5 rounded-lg text-text-muted hover:text-text-secondary hover:bg-surface-hover transition-colors" onclick={onclose} aria-label="Close drawer">
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
</style>
