<script lang="ts">
  import { Plus, Minus, ArrowRight, Clock, Globe, Monitor } from 'lucide-svelte';
  import { ActionBadge, Drawer } from '$shared/ui';
  import { formatDateInJakarta, formatTimeInJakarta, formatDateTimeInJakarta, JAKARTA_OFFSET_MS } from '$shared/utils/jakartaTime';
  import { labels, t } from '$shared/i18n';

  let {
    selectedLog = null,
    drawerOpen = $bindable(false),
    onclose = () => {},
  }: {
    selectedLog?: any;
    drawerOpen?: boolean;
    onclose?: () => void;
  } = $props();

  function getChanges(log: any) {
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

  function getDiffDescription(change: { key: string; old: any; new: any }) {
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
    if (v === 'CREATE') return labels.actionCreated;
    if (v === 'UPDATE') return labels.actionUpdated;
    if (v === 'DELETE') return labels.actionDeleted;
    if (v === 'LOGIN') return labels.login;
    if (v === 'LOGOUT') return labels.logout;
    if (v === 'SHIFT_OPENED') return 'Shift Opened';
    if (v === 'SHIFT_CLOSED') return 'Shift Closed';
    if (v === 'PAYMENT.CREATED') return 'Payment Created';
    return action;
  }

  function getResourceLabel(entityType: string) {
    const map: Record<string, string> = {
      auth: 'Authentication',
      user: labels.user,
      role: labels.role,
      product: labels.product,
      sale: 'Sale',
      category: labels.category,
      brand: labels.brand,
      stock: labels.stock,
      uom: 'Unit of Measure',
      customer: labels.customer,
    };
    return map[entityType] || entityType;
  }

  const fieldLabels: Record<string, string> = {
    name: labels.name,
    username: labels.username,
    email: labels.email,
    role: labels.role,
    role_id: labels.role,
    is_active: `${labels.status} ${labels.active}`,
    is_system: `${labels.system} ${labels.role}`,
    description: labels.description,
    price: labels.price,
    stock: labels.stock,
    category: labels.category,
    category_id: labels.category,
    barcode: labels.barcode,
    sku: labels.sku,
    quantity_change: labels.quantityChange,
    notes: labels.notes,
    invoice_number: labels.invoiceNumber,
    status: labels.status,
    payment_method: labels.paymentMethod,
    discount: labels.discount,
    tax: 'Tax',
    subtotal: labels.subtotal,
    total: labels.total,
    cashier: labels.cashierLabel,
    store: labels.store,
    store_id: labels.store,
    brand: labels.brand,
    brand_id: labels.brand,
    slug: 'Slug',
    parent_id: 'Parent',
    sort_order: 'Sort Order',
    image_url: 'Image URL',
    expiry_date: labels.expireDate,
    unit: 'Unit',
    weight: labels.previewColWeight,
    created_at: labels.createdAt,
    updated_at: labels.updatedAt,
    old_password: 'Old Password',
    new_password: 'New Password',
    permission_ids: labels.permissions,
    permission_id: 'Permission',
  };

  function getFieldLabel(key: string) {
    return fieldLabels[key] || key.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
  }

  function formatTimestamp(d: string | null | undefined) {
    if (!d) return { date: '—', time: '', full: '—' };
    const dateStr = formatDateInJakarta(d);
    const timeStr = formatTimeInJakarta(d);
    return { date: dateStr, time: timeStr, full: `${dateStr} ${timeStr}` };
  }

  function formatDateHuman(d: string | null | undefined) {
    if (!d) return '—';
    const dateObj = new Date(d);
    const nowMs = Date.now() + JAKARTA_OFFSET_MS;
    const shiftedDate = new Date(dateObj.getTime() + JAKARTA_OFFSET_MS);
    const diffMs = nowMs - shiftedDate.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return labels.justNow;
    if (diffMins < 60) return t('minutesAgo', { n: diffMins });
    if (diffHours < 24) return t('hoursAgo', { n: diffHours });
    if (diffDays < 7) return t('daysAgo', { n: diffDays });
    return formatDateInJakarta(d);
  }

  function formatValue(val: any): string {
    if (val == null) return '—';
    if (typeof val === 'boolean') return val ? labels.yes : labels.no;
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
        if (val.length === 0) return labels.none;
        return val.map((v: any) => formatValue(v)).join(', ');
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
      const pairs: string[] = Object.entries(val)
        .filter(([k]: [string, any]) => k !== 'created_at' && k !== 'updated_at')
        .map(([k, v]: [string, any]) => `${getFieldLabel(k)}: ${formatValue(v)}`);
      return pairs.join(', ') || '—';
    }
    return String(val);
  }
</script>

<Drawer bind:open={drawerOpen} width={520} ariaLabel={`${labels.auditLogs} ${labels.details}`} onclose={() => onclose()}>
  {#if selectedLog}
    {@const changes = getChanges(selectedLog)}
    <div class="flex items-center gap-3 mb-4">
      <ActionBadge action={selectedLog.action} />
      <span class="font-mono text-sm text-text-muted bg-surface-default px-2 py-0.5 rounded border border-border/50">{selectedLog.entity_type}</span>
    </div>

    <div class="space-y-5">
      <!-- Human-friendly summary -->
      <div class="bg-surface-default rounded-lg p-4 border border-border/50">
        <p class="text-sm text-text-primary leading-relaxed">
          <span class="font-semibold">{selectedLog.username || `${labels.unknown} ${labels.user}`}</span>
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
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">{labels.description}</p>
          <p class="text-sm text-text-primary">{selectedLog.description}</p>
        </div>
      {/if}

      <!-- Meta grid -->
      <div class="grid grid-cols-2 gap-3">
        <div class="bg-surface-default rounded-lg p-3 border border-border/50">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">{labels.when}</p>
          <p class="text-sm text-text-primary">{formatTimestamp(selectedLog.created_at).full}</p>
          <p class="text-xs text-text-muted mt-0.5">{formatDateHuman(selectedLog.created_at)}</p>
        </div>
        <div class="bg-surface-default rounded-lg p-3 border border-border/50">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">{labels.actor}</p>
          <div class="flex items-center gap-2">
            {#if selectedLog.username && selectedLog.username !== '—'}
              <div class="w-5 h-5 rounded-full gradient-bg-primary flex items-center justify-center shrink-0">
                <span class="text-[8px] font-bold text-white">{selectedLog.username.charAt(0).toUpperCase()}</span>
              </div>
            {/if}
            <p class="text-sm text-text-primary">{selectedLog.username || labels.unknown}</p>
          </div>
          {#if selectedLog.role}
            <p class="text-xs text-text-secondary mt-0.5 capitalize">{selectedLog.role}</p>
          {/if}
        </div>
        <div class="bg-surface-default rounded-lg p-3 border border-border/50">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">{labels.from}</p>
          <div class="flex items-center gap-1.5 text-sm text-text-primary">
            <Globe size={14} class="text-text-muted" />
            <span class="font-mono">{selectedLog.ip_address || '—'}</span>
          </div>
        </div>
        <div class="bg-surface-default rounded-lg p-3 border border-border/50">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">{labels.resource}</p>
          <p class="text-sm text-text-primary capitalize">{getResourceLabel(selectedLog.entity_type) || '—'}</p>
          {#if selectedLog.entity_id}
            <p class="text-xs text-text-secondary font-mono mt-0.5">ID: {selectedLog.entity_id}</p>
          {/if}
        </div>
        <div class="bg-surface-default rounded-lg p-3 border border-border/50">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">{labels.store}</p>
          <p class="text-sm text-text-primary">{selectedLog.store_name || (selectedLog.store_id ? String(selectedLog.store_id) : '—')}</p>
        </div>
      </div>

      <!-- User Agent -->
      {#if selectedLog.user_agent}
        <div>
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-2">{labels.browserDevice}</p>
          <div class="flex items-start gap-2 p-3 bg-surface-default rounded-lg border border-border/50">
            <Monitor size={14} class="text-text-muted mt-0.5 shrink-0" />
            <p class="text-xs text-text-secondary font-mono leading-relaxed break-all">{selectedLog.user_agent}</p>
          </div>
        </div>
      {/if}

      <!-- Data Changes -->
      {#if changes.length > 0}
        <div>
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-3">{labels.whatChanged}</p>
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
  {/if}
</Drawer>

<style>
  :global(input[type="date"]::-webkit-calendar-picker-indicator) {
    filter: invert(1);
    cursor: pointer;
  }
</style>
