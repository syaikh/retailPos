<script lang="ts">
  import { Badge, Button, Skeleton } from '$shared/ui';
  import { Pencil, Trash2, Check, X, Search } from 'lucide-svelte';

  let {
    customers = [],
    loading = false,
    searchQuery = '',
    canUpdate = false,
    canDelete = false,
    selectedIds = $bindable(new Set<number>()),
    editingId = $bindable(null as number | null),
    editName = $bindable(''),
    editPhone = $bindable(''),
    editEmail = $bindable(''),
    editActive = $bindable(true),
    sortBy = $bindable('name'),
    sortDir = $bindable('asc'),
    onselectall = () => {},
    onselect = (id: number) => {},
    onsort = (col: string) => {},
    onedit = (c: any) => {},
    oncanceledit = () => {},
    onsaveedit = (id: number) => {},
    ondeactivate = (c: any) => {},
  }: {
    customers: any[];
    loading: boolean;
    searchQuery: string;
    canUpdate: boolean;
    canDelete: boolean;
    selectedIds: Set<number>;
    editingId: number | null;
    editName: string;
    editPhone: string;
    editEmail: string;
    editActive: boolean;
    sortBy: string;
    sortDir: string;
    onselectall?: () => void;
    onselect?: (id: number) => void;
    onsort?: (col: string) => void;
    onedit?: (c: any) => void;
    oncanceledit?: () => void;
    onsaveedit?: (id: number) => void;
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

<table class="min-w-full text-sm table-fixed">
  <thead class="bg-muted/50">
    <tr>
      <th class="p-4 font-semibold w-12">
        <input type="checkbox" class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary" checked={allSelected} bind:indeterminate={someSelected} onchange={toggleSelectAll} aria-label="Select all customers" />
      </th>
      <th class="text-left p-4 font-semibold w-[26%]">
        <button type="button" class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => onsort('name')}>
          NAME {#if sortBy === 'name'}<span>{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
        </button>
      </th>
      <th class="text-left p-4 font-semibold w-[18%]">
        <button type="button" class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => onsort('phone')}>
          PHONE {#if sortBy === 'phone'}<span>{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
        </button>
      </th>
      <th class="text-left p-4 font-semibold w-[26%]">
        <button type="button" class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => onsort('email')}>
          EMAIL {#if sortBy === 'email'}<span>{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
        </button>
      </th>
      <th class="text-left p-4 font-semibold w-[14%]">
        <button type="button" class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => onsort('status')}>
          STATUS {#if sortBy === 'status'}<span>{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
        </button>
      </th>
      <th class="text-center p-4 font-semibold w-20">Actions</th>
    </tr>
  </thead>
  <tbody>
    {#if loading}
      {#each { length: 5 } as _, i}
        <tr class="border-t border-border">
          <td class="px-4 py-3" colspan={6}>
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
        <td colspan={6}>
          <div class="px-4 py-16 text-center">
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
        {#if editingId === c.id}
          <tr class="border-t border-border bg-primary-subtle/10">
            <td class="px-4 py-1.5 h-12 overflow-hidden">
              <div class="flex items-center gap-2">
                <div class="w-7 h-7 rounded-full bg-primary-subtle text-primary-light flex items-center justify-center text-[10px] font-bold shrink-0">
                  {getInitials(editName)}
                </div>
                <input class="flex-1 h-6 px-2.5 py-1 text-xs leading-none rounded-lg bg-bg-secondary text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-primary-default/20 border-0" bind:value={editName} aria-label="Edit name" />
              </div>
            </td>
            <td class="px-4 py-1.5 h-12 overflow-hidden"><input class="w-full h-6 px-2.5 py-1 text-xs leading-none rounded-lg bg-bg-secondary text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-primary-default/20 border-0" bind:value={editPhone} aria-label="Edit phone" /></td>
            <td class="px-4 py-1.5 h-12 overflow-hidden"><input class="w-full h-6 px-2.5 py-1 text-xs leading-none rounded-lg bg-bg-secondary text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-primary-default/20 border-0" bind:value={editEmail} aria-label="Edit email" /></td>
            <td class="px-4 py-1.5 h-12 overflow-hidden">
              <label class="flex items-center gap-2 text-xs">
                <input type="checkbox" bind:checked={editActive} />
                {editActive ? 'Active' : 'Inactive'}
              </label>
            </td>
            <td class="px-4 py-1.5 h-12 overflow-hidden">
              <div class="flex items-center gap-1">
                <Button variant="ghost" size="icon" class="text-text-muted hover:text-primary-light transition-all active:scale-90" onclick={() => onsaveedit(c.id)} title="Save" aria-label="Save">
                  <Check size={14} />
                </Button>
                <Button variant="ghost" size="icon" class="text-text-muted hover:text-danger transition-all active:scale-90" onclick={oncanceledit} title="Cancel" aria-label="Cancel">
                  <X size={14} />
                </Button>
              </div>
            </td>
          </tr>
        {:else}
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
        {/if}
      {/each}
    {/if}
  </tbody>
</table>
