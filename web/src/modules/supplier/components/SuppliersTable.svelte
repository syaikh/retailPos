<script lang="ts">
  import { Button, Skeleton, SortableHeader, Badge, Tooltip, Dropdown } from '$shared/ui';
  import { labels, t, formatLocaleDate } from '$shared/i18n';
  import { formatLocaleDateInJakarta } from '$shared/utils/jakartaTime';
  import { Pencil, Trash2, Truck, Copy, Package, MoreVertical } from 'lucide-svelte';
  import type { Supplier } from '../types';

  function timeAgo(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    const now = new Date();
    const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);
    if (seconds < 60) return 'just now';
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    if (days < 30) return `${days}d ago`;
    const months = Math.floor(days / 30);
    if (months < 12) return `${months}mo ago`;
    const years = Math.floor(months / 12);
    return `${years}y ago`;
  }

  function formatDateTime(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    return formatLocaleDateInJakarta(dateStr, {
      day: 'numeric', month: 'short', year: 'numeric',
      hour: '2-digit', minute: '2-digit',
    });
  }

  let {
    suppliers = [],
    loading = false,
    searchQuery = '',
    canEdit = false,
    canDelete = false,
    canCreate = false,
    sortBy = 'name',
    sortDir = 'asc',
    onsort = () => {},
    onedit = () => {},
    ondelete = () => {},
    onduplicate = () => {},
    onviewproducts = () => {},
    onrowclick = () => {},
    onbulkactivate = () => {},
    onbulkdeactivate = () => {},
    onbulkdelete = () => {},
  }: {
    suppliers?: Supplier[];
    loading?: boolean;
    searchQuery?: string;
    canEdit?: boolean;
    canDelete?: boolean;
    canCreate?: boolean;
    sortBy?: string;
    sortDir?: 'asc' | 'desc';
    onsort?: (col: string) => void;
    onedit?: (supplier: Supplier) => void;
    ondelete?: (supplier: Supplier) => void;
    onduplicate?: (supplier: Supplier) => void;
    onviewproducts?: (supplier: Supplier) => void;
    onrowclick?: (supplier: Supplier) => void;
    onbulkactivate?: (ids: number[]) => void;
    onbulkdeactivate?: (ids: number[]) => void;
    onbulkdelete?: (ids: number[]) => void;
  } = $props();

  let selectedIds = $state<number[]>([]);
  let openMenuId = $state<number | null>(null);

  let allSelected = $derived(
    suppliers.length > 0 && selectedIds.length === suppliers.length
  );
  let someSelected = $derived(
    selectedIds.length > 0 && selectedIds.length < suppliers.length
  );

  function toggleAll() {
    if (allSelected) {
      selectedIds = [];
    } else {
      selectedIds = suppliers.map(s => s.id);
    }
  }

  function toggleOne(id: number) {
    if (selectedIds.includes(id)) {
      selectedIds = selectedIds.filter(i => i !== id);
    } else {
      selectedIds = [...selectedIds, id];
    }
  }

  function handleActionKeydown(e: KeyboardEvent, action: () => void) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      action();
    }
  }
</script>

{#if loading}
  <div class="overflow-x-auto">
    <table class="w-full" style="table-layout: fixed;" aria-busy="true" aria-label={labels.loadingSuppliers}>
      <colgroup>
        <col style="width: 3%;" />
        <col style="width: 22%;" />
        <col style="width: 18%;" />
        <col style="width: 15%;" />
        <col style="width: 20%;" />
        <col style="width: 10%;" />
        <col style="width: 12%;" />
      </colgroup>
      <thead><tr><th></th><th>{labels.name}</th><th>{labels.contactPerson}</th><th>{labels.phone}</th><th>{labels.email}</th><th>{labels.statusLabel}</th><th></th></tr></thead>
      <tbody>{#each Array(5) as _}<tr>{#each Array(7) as _}<td><Skeleton class="h-4 w-20" /></td>{/each}</tr>{/each}</tbody>
    </table>
  </div>
{:else if suppliers.length === 0}
  <div class="flex flex-col items-center justify-center py-12 text-text-muted" role="status">
    <Truck class="w-12 h-12 mb-3" aria-hidden="true" />
    <p>{labels.noSuppliersFound}</p>
    {#if searchQuery}
      <p class="text-sm text-text-muted mt-1">{labels.tryAdjustingYourSearch}</p>
    {/if}
  </div>
{:else}
  <div class="overflow-x-auto">
    <table class="w-full min-w-[800px]" style="table-layout: fixed;" role="grid" aria-label={labels.suppliers}>
      <colgroup>
        <col style="width: 3%;" />
        <col style="width: 20%;" />
        <col style="width: 16%;" />
        <col style="width: 13%;" />
        <col style="width: 18%;" />
        <col style="width: 9%;" />
        <col style="width: 9%;" />
        <col style="width: 12%;" />
      </colgroup>
      <thead class="bg-muted/50">
        <tr class="border-b text-left text-sm text-text-muted">
          <th class="px-4 py-3" scope="col">
            <input
              type="checkbox"
              class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary"
              checked={allSelected}
              bind:indeterminate={someSelected}
              onchange={toggleAll}
              aria-label={labels.selectAllSuppliers}
            />
          </th>
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label={labels.nameLabel} column="name" sortColumn={sortBy} sortDirection={sortDir} onsort={onsort} />
          </th>
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label={labels.contactPerson} column="contact_name" sortColumn={sortBy} sortDirection={sortDir} onsort={onsort} />
          </th>
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label={labels.phoneLabel} column="phone" sortColumn={sortBy} sortDirection={sortDir} onsort={onsort} />
          </th>
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label={labels.emailLabel} column="email" sortColumn={sortBy} sortDirection={sortDir} onsort={onsort} />
          </th>
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label={labels.statusLabel} column="is_active" sortColumn={sortBy} sortDirection={sortDir} onsort={onsort} />
          </th>
          <th class="px-4 py-3 font-semibold" scope="col">{labels.consignmentFilter}</th>
          <th class="px-4 py-3 font-semibold text-right" scope="col">{labels.actionsLabel}</th>
        </tr>
      </thead>
      <tbody>
        {#each suppliers as supplier (supplier.id)}
          <tr
            class="border-b border-border transition-colors hover:bg-muted/50 {selectedIds.includes(supplier.id) ? 'bg-muted/30' : ''} cursor-pointer"
            onclick={() => onrowclick(supplier)}
            onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onrowclick(supplier); } }}
            role="button"
            tabindex="0"
          >
            <td class="px-4 py-3" onclick={(e) => e.stopPropagation()}>
              <input
                type="checkbox"
                class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary"
                checked={selectedIds.includes(supplier.id)}
                onchange={() => toggleOne(supplier.id)}
                aria-label={t('selectSupplierWithName', { name: supplier.name })}
              />
            </td>
            <td class="px-4 py-3 font-medium truncate">
              <Tooltip content={supplier.name}>
                {supplier.name}
              </Tooltip>
            </td>
            <td class="px-4 py-3 text-text-secondary truncate">{supplier.contact_name || '-'}</td>
            <td class="px-4 py-3 text-text-secondary tabular-nums">{supplier.phone || '-'}</td>
            <td class="px-4 py-3 text-text-secondary text-sm truncate">{supplier.email || '-'}</td>
            <td class="px-4 py-3">
              <Badge variant={supplier.is_active ? 'success' : 'muted'}>
                {supplier.is_active ? labels.active : labels.inactive}
              </Badge>
            </td>
            <td class="px-4 py-3">
              {#if supplier.is_consignment}
                <Badge variant="warning">{labels.consignmentFilter}</Badge>
              {/if}
            </td>
            <td class="px-4 py-3 text-right" onclick={(e) => e.stopPropagation()}>
              <div class="flex items-center justify-center" role="group" aria-label={t('actionsFor', { name: supplier.name })}>
                <Dropdown placement="bottom-end" items={[]}>
                  {#snippet content({ close })}
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onviewproducts(supplier); close(); }}>
                      <Package size={14} /> {labels.viewProducts}
                    </button>
                    {#if canEdit}
                      <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onedit(supplier); close(); }}>
                        <Pencil size={14} /> {labels.edit}
                      </button>
                    {/if}
                    {#if canCreate}
                      <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onduplicate(supplier); close(); }}>
                        <Copy size={14} /> {labels.duplicate}
                      </button>
                    {/if}
                    {#if canDelete}
                      <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-danger hover:bg-danger-subtle transition-colors" role="menuitem" onclick={() => { ondelete(supplier); close(); }}>
                        <Trash2 size={14} /> {labels.delete}
                      </button>
                    {/if}
                  {/snippet}
                  {#snippet trigger({ toggle })}
                    <Button variant="ghost" size="icon" class="text-text-muted hover:text-text-primary" onclick={(e) => { e.stopPropagation(); toggle(); }} aria-label={t('actionsFor', { name: supplier.name })}>
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

  {#if selectedIds.length > 0}
    <div class="px-4 py-3 bg-primary-subtle/20 border-t border-primary-default/20 flex items-center gap-3">
      <span class="text-sm text-text-secondary">{t('selectedCountLabel', { count: selectedIds.length })}</span>
      <Button variant="secondary" size="sm" onclick={() => onbulkactivate(selectedIds)}>{labels.activate}</Button>
      <Button variant="secondary" size="sm" onclick={() => onbulkdeactivate(selectedIds)}>{labels.deactivate}</Button>
      <Button variant="danger" size="sm" onclick={() => onbulkdelete(selectedIds)}>{labels.delete}</Button>
    </div>
  {/if}
{/if}