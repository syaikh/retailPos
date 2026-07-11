<script lang="ts">
  import { Badge, Button, Skeleton, SortableHeader } from '$shared/ui';
  import { Pencil, Trash2, Search } from 'lucide-svelte';

  let {
    customers = [],
    loading = false,
    searchQuery = '',
    canUpdate = false,
    canDelete = false,
    selectedIds = $bindable(new Set<number>()),
    sortBy = $bindable('name'),
    sortDir = $bindable('asc'),
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
    sortDir: string;
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
<table class="min-w-full text-sm table-fixed min-w-[800px]">
  <thead class="bg-muted/50">
    <tr>
      <th class="p-4 font-semibold w-12">
        <input type="checkbox" class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary" checked={allSelected} bind:indeterminate={someSelected} onchange={toggleSelectAll} aria-label="Select all customers" />
      </th>
      <th class="text-left p-4 font-semibold w-[20%]">
        <SortableHeader label="NAME" column="name" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
      </th>
      <th class="text-left p-4 font-semibold w-[12%]">
        <SortableHeader label="PHONE" column="phone" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
      </th>
      <th class="text-left p-4 font-semibold w-[16%]">
        <SortableHeader label="EMAIL" column="email" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
      </th>
      <th class="text-left p-4 font-semibold w-[16%]">ADDRESS</th>
      <th class="text-left p-4 font-semibold w-[14%]">NOTE</th>
      <th class="text-left p-4 font-semibold w-[12%]">
        <SortableHeader label="STATUS" column="status" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
      </th>
      <th class="text-center p-4 font-semibold w-20">Actions</th>
    </tr>
  </thead>
  <tbody>
    {#if loading}
      {#each { length: 5 } as _, i}
        <tr class="border-t border-border" aria-hidden="true">
          <td class="px-4 py-3" colspan={8}>
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
        <td colspan={8}>
          <div class="px-4 py-16 text-center" role="status">
            <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
              <Search size={32} class="text-text-muted" />
            </div>
            <p class="text-text-primary font-semibold mt-4">
              {searchQuery ? 'No customers found' : 'No customers yet'}
            </p>
            <p class="text-text-muted text-sm mt-1">
              {searchQuery ? `No customers matching "${searchQuery}"` : 'Start by adding your first customer'}
            </p>
          </div>
        </td>
      </tr>
    {:else}
      {#each customers as c}
        <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
          <td class="px-4 py-1.5 h-12 w-12" onclick={(e) => e.stopPropagation()}>
            <input type="checkbox" class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary" checked={selectedIds.has(c.id)} onchange={() => toggleSelect(c.id)} aria-label="Select {c.name}" />
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
            {#if c.note}
              <span class="truncate block">{c.note}</span>
            {:else}
              <span class="text-text-muted">—</span>
            {/if}
          </td>
          <td class="px-4 py-1.5 h-12 overflow-hidden">
            {#if c.is_active !== false}
              <Badge variant="success" size="sm">Active</Badge>
            {:else}
              <Badge variant="danger" size="sm">Inactive</Badge>
            {/if}
          </td>
          <td class="px-4 py-1.5 h-12 overflow-hidden text-center">
            <div class="flex items-center justify-center gap-1">
              {#if canUpdate}
                <Button variant="ghost" size="icon" class="text-text-muted hover:text-primary-light transition-all active:scale-90" onclick={() => onedit(c)} title="Edit" aria-label="Edit">
                  <Pencil size={14} />
                </Button>
              {/if}
              {#if canDelete && c.is_active !== false}
                <Button variant="ghost" size="icon" class="text-text-muted hover:text-danger hover:bg-danger-subtle transition-all active:scale-90" onclick={() => ondeactivate(c)} title="Deactivate" aria-label="Deactivate">
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
