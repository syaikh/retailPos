<script lang="ts">
  import { Badge, Button, Skeleton, SortableHeader } from '$shared/ui';
  import { Pencil, Trash2, Search } from 'lucide-svelte';
  import { labels, t } from '$shared/i18n';

  let {
    customers = [],
    loading = false,
    searchQuery = '',
    canUpdate = false,
    canDelete = false,
    selectedIds = $bindable(new Set<number>()),
    sortBy = $bindable('name'),
    sortDir = $bindable<'asc' | 'desc'>('asc'),
    onselectall = () => {},
    onselect = (id: number) => {},
    onsort = (col: string) => {},
    onedit = (c: any) => {},
    ondeactivate = (c: any) => {},
  }: {
    customers: any[];
    loading: boolean;
    searchQuery: string;
    canUpdate: boolean;
    canDelete: boolean;
    selectedIds: Set<number>;
    sortBy: string;
    sortDir: 'asc' | 'desc';
    onselectall?: () => void;
    onselect?: (id: number) => void;
    onsort?: (col: string) => void;
    onedit?: (c: any) => void;
    ondeactivate?: (c: any) => void;
  } = $props();

  let allSelected = $derived(customers.length > 0 && customers.every(c => selectedIds.has(c.id)));
  let someSelected = $derived(selectedIds.size > 0 && !allSelected);

  function toggleSelectAll() {
    if (allSelected) {
      selectedIds = new Set();
    } else {
      selectedIds = new Set(customers.map(c => c.id));
    }
    onselectall();
  }

  function toggleSelect(id: number) {
    const next = new Set(selectedIds);
    if (next.has(id)) { next.delete(id); } else { next.add(id); }
    selectedIds = next;
    onselect(id);
  }

  function getInitials(name: string): string {
    if (!name) return '?';
    const parts = name.trim().split(/\s+/);
    if (parts.length >= 2) {
      return (parts[0].charAt(0) + parts[1].charAt(0)).toUpperCase();
    }
    return parts[0].charAt(0).toUpperCase();
  }
</script>

<div class="overflow-x-auto">
<table class="min-w-full text-sm table-fixed min-w-[900px]">
  <thead class="bg-muted/50">
    <tr>
      <th class="p-4 font-semibold w-12">
        <input type="checkbox" class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary" checked={allSelected} bind:indeterminate={someSelected} onchange={toggleSelectAll} aria-label={labels.selectAllCustomers} />
      </th>
      <th class="text-left p-4 font-semibold w-[18%]">
        <SortableHeader label={labels.nameLabel} column="name" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
      </th>
      <th class="text-left p-4 font-semibold w-[11%]">
        <SortableHeader label={labels.phoneLabel} column="phone" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
      </th>
      <th class="text-left p-4 font-semibold w-[14%]">
        <SortableHeader label={labels.emailLabel} column="email" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
      </th>
      <th class="text-left p-4 font-semibold w-[14%]">{labels.addressLabel}</th>
      <th class="text-left p-4 font-semibold w-[11%]">
        <SortableHeader label={labels.groupLabel} column="group" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
      </th>
      <th class="text-left p-4 font-semibold w-[11%]">{labels.notesLabel}</th>
      <th class="text-left p-4 font-semibold w-[10%]">
        <SortableHeader label={labels.statusLabel} column="status" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
      </th>
      <th class="text-center p-4 font-semibold w-20">{labels.actionsLabel}</th>
    </tr>
  </thead>
  <tbody>
    {#if loading}
      {#each { length: 5 } as _, i}
        <tr class="border-t border-border" aria-hidden="true">
          <td class="px-4 py-3" colspan={9}>
            <div class="flex items-center gap-3">
              <Skeleton width="w-8" height="h-8" rounded="rounded-full" />
              <div class="flex-1 space-y-2">
                <Skeleton width="w-3/5" height="h-4" />
                <Skeleton width="w-2/5" height="h-4" />
              </div>
            </div>
          </td>
        </tr>
      {/each}
    {:else if customers.length === 0}
      <tr class="border-t border-border">
        <td colspan={9}>
          <div class="px-4 py-16 text-center" role="status" aria-live="polite">
            <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
              <Search size={32} class="text-text-muted" />
            </div>
            <p class="text-text-primary font-semibold mt-4">
              {searchQuery ? labels.noCustomersFound : labels.noCustomersYet}
            </p>
            <p class="text-text-muted text-sm mt-1">
              {searchQuery ? t('noCustomersMatching', { query: searchQuery }) : labels.addFirstCustomer}
            </p>
          </div>
        </td>
      </tr>
    {:else}
      {#each customers as c}
        <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
          <td class="px-4 py-1.5 h-12 w-12" onclick={(e) => e.stopPropagation()}>
            <input type="checkbox" class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary" checked={selectedIds.has(c.id)} onchange={() => toggleSelect(c.id)} aria-label={t('selectCustomerWithName', { name: c.name })} />
          </td>
          <td class="px-4 py-1.5 h-12 overflow-hidden">
            <div class="flex items-center gap-3">
              <div class="w-8 h-8 rounded-full bg-primary-subtle text-primary-light flex items-center justify-center text-xs font-bold shrink-0">
                {getInitials(c.name)}
              </div>
              <span class="truncate">{c.name}</span>
            </div>
          </td>
          <td class="px-4 py-1.5 h-12 overflow-hidden">{c.phone || '—'}</td>
          <td class="px-4 py-1.5 h-12 overflow-hidden">{c.email || '—'}</td>
          <td class="px-4 py-1.5 h-12 overflow-hidden">
            {#if c.address}
              <span class="truncate block">{c.address}</span>
            {:else}
              <span class="text-text-muted">—</span>
            {/if}
          </td>
          <td class="px-4 py-1.5 h-12 overflow-hidden">
            {#if c.customer_group_name}
              <Badge variant="primary" size="sm">{c.customer_group_name}</Badge>
            {:else}
              <span class="text-text-muted">—</span>
            {/if}
          </td>
          <td class="px-4 py-1.5 h-12 overflow-hidden">
            {#if c.note}
              <span class="truncate block">{c.note}</span>
            {:else}
              <span class="text-text-muted">—</span>
            {/if}
          </td>
          <td class="px-4 py-1.5 h-12 overflow-hidden">
            {#if c.is_active !== false}
              <Badge variant="success" size="sm">{labels.active}</Badge>
            {:else}
              <Badge variant="danger" size="sm">{labels.inactive}</Badge>
            {/if}
          </td>
          <td class="px-4 py-1.5 h-12 overflow-hidden text-center">
            <div class="flex items-center justify-center gap-1">
              {#if canUpdate}
                <Button variant="ghost" size="icon" class="text-text-muted hover:text-primary-light transition-all active:scale-90" onclick={() => onedit(c)} title={labels.edit} aria-label={labels.edit}>
                  <Pencil size={14} />
                </Button>
              {/if}
              {#if canDelete && c.is_active !== false}
                <Button variant="ghost" size="icon" class="text-text-muted hover:text-danger hover:bg-danger-subtle transition-all active:scale-90" onclick={() => ondeactivate(c)} title={labels.deactivate} aria-label={labels.deactivate}>
                  <Trash2 size={14} />
                </Button>
              {/if}
            </div>
          </td>
        </tr>
      {/each}
    {/if}
  </tbody>
</table>
</div>
