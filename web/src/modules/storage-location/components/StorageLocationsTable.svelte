<script lang="ts">
  import { Badge, Button, Skeleton, SortableHeader, Tooltip, Dropdown } from '$shared/ui';
  import { Search, MoreVertical, Pencil, Trash2, Power, PowerOff, Warehouse as WarehouseIcon, Store as StoreIcon } from 'lucide-svelte';
  import { labels, formatLocaleDate } from '$shared/i18n';
  import type { StorageLocation } from '../types';

  let {
    locations = [],
    loading = false,
    searchQuery = '',
    warehouseMap = {} as Record<number, string>,
    storeMap = {} as Record<number, string>,
    canUpdate = false,
    canDelete = false,
    canCreate = false,
    sortBy = $bindable('code'),
    sortDir = $bindable<'asc' | 'desc'>('asc'),
    onsort = (col: string) => {},
    onedit = (_g: StorageLocation) => {},
    ondelete = (_g: StorageLocation) => {},
    onbulkactivate = (_ids: number[]) => {},
    onbulkdeactivate = (_ids: number[]) => {},
    onbulkdelete = (_ids: number[]) => {},
  }: {
    locations: StorageLocation[];
    loading: boolean;
    searchQuery: string;
    warehouseMap: Record<number, string>;
    storeMap: Record<number, string>;
    canUpdate: boolean;
    canDelete: boolean;
    canCreate: boolean;
    sortBy: string;
    sortDir: 'asc' | 'desc';
    onsort?: (col: string) => void;
    onedit?: (g: StorageLocation) => void;
    ondelete?: (g: StorageLocation) => void;
    onbulkactivate?: (ids: number[]) => void;
    onbulkdeactivate?: (ids: number[]) => void;
    onbulkdelete?: (ids: number[]) => void;
  } = $props();

  let selectedIds = $state<Set<number>>(new Set());

  let allSelected = $derived(locations.length > 0 && locations.every(g => selectedIds.has(g.id)));
  let someSelected = $derived(locations.some(g => selectedIds.has(g.id)) && !allSelected);
  let selectedCount = $derived(selectedIds.size);

  function toggleSelect(id: number) {
    const next = new Set(selectedIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    selectedIds = next;
  }

  function toggleSelectAll() {
    if (allSelected) {
      selectedIds = new Set();
    } else {
      selectedIds = new Set(locations.map(g => g.id));
    }
  }

  function clearSelection() {
    selectedIds = new Set();
  }

  function scopeLabel(g: StorageLocation): { label: string; type: 'warehouse' | 'store' } | null {
    if (g.warehouse_id != null && warehouseMap[g.warehouse_id]) {
      return { label: warehouseMap[g.warehouse_id], type: 'warehouse' };
    }
    if (g.store_id != null && storeMap[g.store_id]) {
      return { label: storeMap[g.store_id], type: 'store' };
    }
    return null;
  }

  function timeAgo(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    const now = Date.now();
    const then = new Date(dateStr).getTime();
    const diff = now - then;
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return labels.justNow;
    if (mins < 60) return labels.minutesAgo.replace('{n}', String(mins));
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return labels.hoursAgo.replace('{n}', String(hrs));
    const days = Math.floor(hrs / 24);
    if (days < 30) return labels.daysAgo.replace('{n}', String(days));
    const months = Math.floor(days / 30);
    return labels.monthsAgo.replace('{n}', String(months));
  }

  function formatDateTime(dateStr: string | undefined): string {
    if (!dateStr) return '';
    try {
      return formatLocaleDate(new Date(dateStr), { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
    } catch { return dateStr; }
  }
</script>

{#if loading}
  <div class="overflow-x-auto">
  <table class="w-full" style="table-layout: fixed;" aria-busy="true" aria-label={labels.loadingStorageLocations}>
    <colgroup>
      <col style="width: 3%;" />
      <col style="width: 15%;" />
      <col style="width: 24%;" />
      <col style="width: 20%;" />
      <col style="width: 18%;" />
      <col style="width: 10%;" />
      <col style="width: 8%;" />
    </colgroup>
    <thead class="bg-muted/50">
      <tr class="border-b text-left text-xs text-text-muted">
        <th class="px-3 py-3"></th>
        <th class="px-4 py-3 font-semibold">{labels.codeLabel}</th>
        <th class="px-4 py-3 font-semibold">{labels.nameLabel}</th>
        <th class="px-4 py-3 font-semibold">{labels.scopeLabel}</th>
        <th class="px-4 py-3 font-semibold">{labels.statusLabel}</th>
        <th class="px-4 py-3 font-semibold">{labels.updatedAtLabel}</th>
        <th class="px-4 py-3 font-semibold text-center">{labels.actionLabel}</th>
      </tr>
    </thead>
    <tbody>
      {#each { length: 5 } as _}
        <tr class="border-t border-border" aria-hidden="true">
          <td class="px-3 py-4"><Skeleton class="h-4 w-4" /></td>
          <td class="px-4 py-4"><Skeleton class="h-4 w-14" /></td>
          <td class="px-4 py-4"><Skeleton class="h-4 w-3/5" /></td>
          <td class="px-4 py-4"><Skeleton class="h-4 w-24" /></td>
          <td class="px-4 py-4"><Skeleton class="h-5 w-16" /></td>
          <td class="px-4 py-4"><Skeleton class="h-4 w-20" /></td>
          <td class="px-4 py-4"><Skeleton class="h-6 w-6 mx-auto rounded" /></td>
        </tr>
      {/each}
    </tbody>
  </table>
  </div>
{:else if locations.length === 0}
  <div class="flex flex-col items-center justify-center py-12 text-text-muted" role="status" aria-live="polite">
    <Search class="w-12 h-12 mb-3" aria-hidden="true" />
    <p class="text-sm">{searchQuery ? labels.noLocationsFound : labels.noStorageLocationsYet}</p>
    {#if !searchQuery && canCreate}
      <p class="text-xs text-text-muted mt-1">{labels.clickAddLocationToCreateFirst}</p>
    {/if}
  </div>
{:else}
  <div class="overflow-x-auto">
  <table class="w-full min-w-[720px]" style="table-layout: fixed;" aria-label={labels.storageLocations}>
    <colgroup>
      <col style="width: 3%;" />
      <col style="width: 15%;" />
      <col style="width: 24%;" />
      <col style="width: 20%;" />
      <col style="width: 18%;" />
      <col style="width: 10%;" />
      <col style="width: 8%;" />
    </colgroup>
    <thead class="bg-muted/50 sticky top-0 z-10">
      <tr class="border-b text-left text-xs text-text-muted">
        <th class="px-3 py-3">
          <input type="checkbox" class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary" checked={allSelected} bind:indeterminate={someSelected} onchange={toggleSelectAll} aria-label={labels.selectAll} />
        </th>
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label={labels.codeLabel} column="code" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label={labels.nameLabel} column="name" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold">{labels.scopeLabel}</th>
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label={labels.statusLabel} column="status" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label={labels.updatedAtLabel} column="updated_at" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold text-center" scope="col">{labels.actionLabel}</th>
      </tr>
    </thead>
    <tbody>
      {#each locations as g (g.id)}
        <tr class="border-t border-border transition-colors hover:bg-muted/50 {selectedIds.has(g.id) ? 'bg-muted/30' : ''}">
          <td class="px-3 py-4">
            <input type="checkbox" class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary" checked={selectedIds.has(g.id)} onchange={() => toggleSelect(g.id)} aria-label={labels.selectStorageLocation.replace('{name}', g.name)} />
          </td>
          <td class="px-4 py-4">
            <span class="font-mono text-sm text-text-primary">{g.code}</span>
          </td>
          <td class="px-4 py-4">
            <div class="min-w-0">
              <span class="truncate font-medium block text-sm">{g.name}</span>
              {#if g.notes}
                <span class="truncate block text-xs text-text-muted">{g.notes}</span>
              {/if}
            </div>
          </td>
          <td class="px-4 py-4">
            {#if scopeLabel(g)}
              <div class="flex items-center gap-1.5 text-sm text-text-secondary">
                {#if scopeLabel(g)?.type === 'warehouse'}
                  <WarehouseIcon size={14} class="text-text-muted shrink-0" />
                {:else}
                  <StoreIcon size={14} class="text-text-muted shrink-0" />
                {/if}
                <span class="truncate">{scopeLabel(g)?.label}</span>
              </div>
            {:else}
              <span class="text-sm text-text-muted">-</span>
            {/if}
          </td>
          <td class="px-4 py-4">
            {#if g.is_active}
              <Badge variant="success" size="sm">{labels.active}</Badge>
            {:else}
              <Badge variant="danger" size="sm">{labels.inactive}</Badge>
            {/if}
          </td>
          <td class="px-4 py-4 text-sm text-text-muted">
            <Tooltip content={formatDateTime(g.updated_at || g.created_at)} delay={400}>
              <span class="truncate block">{timeAgo(g.updated_at || g.created_at)}</span>
            </Tooltip>
          </td>
          <td class="px-4 py-4">
            <div class="flex items-center justify-center" role="group" aria-label={labels.actionsFor.replace('{name}', g.name)}>
              <Dropdown placement="bottom-end" items={[]}>
                {#snippet content({ close })}
                  {#if canUpdate}
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onedit(g); close(); }}>
                      <Pencil size={14} /> {labels.edit}
                    </button>
                  {/if}
                  {#if canDelete}
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-danger hover:bg-danger-subtle transition-colors" role="menuitem" onclick={() => { ondelete(g); close(); }}>
                      <Trash2 size={14} /> {labels.delete}
                    </button>
                  {/if}
                {/snippet}
                {#snippet trigger({ toggle })}
                  <Button variant="ghost" size="icon" class="text-text-muted hover:text-text-primary" onclick={(e) => { e.stopPropagation(); toggle(); }} aria-label={labels.actionsFor.replace('{name}', g.name)}>
                    <MoreVertical size={16} />
                  </Button>
                {/snippet}
              </Dropdown>
            </div>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
  </div>

  {#if selectedCount > 0}
    <div class="flex items-center justify-between px-4 py-3 bg-primary-subtle/15 border-t border-primary-default/15">
      <span class="text-sm text-text-primary font-medium">{labels.locationsSelected.replace('{count}', String(selectedCount))}</span>
      <div class="flex items-center gap-2">
        {#if canUpdate}
          <Button variant="secondary" size="sm" onclick={() => { onbulkactivate([...selectedIds]); clearSelection(); }}>
            <Power size={14} /> {labels.activate}
          </Button>
          <Button variant="secondary" size="sm" onclick={() => { onbulkdeactivate([...selectedIds]); clearSelection(); }}>
            <PowerOff size={14} /> {labels.deactivate}
          </Button>
        {/if}
        {#if canDelete}
          <Button variant="danger" size="sm" onclick={() => { onbulkdelete([...selectedIds]); clearSelection(); }}>
            <Trash2 size={14} /> {labels.delete}
          </Button>
        {/if}
        <Button variant="ghost" size="sm" onclick={clearSelection}>{labels.cancel}</Button>
      </div>
    </div>
  {/if}
{/if}
