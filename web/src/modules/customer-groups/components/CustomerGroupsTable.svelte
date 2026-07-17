<script lang="ts">
  import { Badge, Button, Skeleton, SortableHeader } from '$shared/ui';
  import { Pencil, Trash2, Search } from 'lucide-svelte';

  let {
    groups = [],
    loading = false,
    searchQuery = '',
    canUpdate = false,
    canDelete = false,
    sortBy = $bindable('name'),
    sortDir = $bindable<'asc' | 'desc'>('asc'),
    onsort = (col: string) => {},
    onedit = (g: any) => {},
    ondelete = (g: any) => {},
  }: {
    groups: any[];
    loading: boolean;
    searchQuery: string;
    canUpdate: boolean;
    canDelete: boolean;
    sortBy: string;
    sortDir: 'asc' | 'desc';
    onsort?: (col: string) => void;
    onedit?: (g: any) => void;
    ondelete?: (g: any) => void;
  } = $props();

  function formatDate(dateStr?: string): string {
    if (!dateStr) return '—';
    try {
      return new Date(dateStr).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' });
    } catch {
      return dateStr;
    }
  }
</script>

<div class="overflow-x-auto">
<table class="min-w-full text-sm table-fixed min-w-[700px]">
  <thead class="bg-muted/50">
    <tr>
      <th class="text-left p-4 font-semibold w-[30%]">
        <SortableHeader label="NAME" column="name" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
      </th>
      <th class="text-left p-4 font-semibold w-[35%]">DESCRIPTION</th>
      <th class="text-left p-4 font-semibold w-[15%]">
        <SortableHeader label="STATUS" column="status" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
      </th>
      <th class="text-left p-4 font-semibold w-[15%]">
        <SortableHeader label="CREATED" column="created_at" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
      </th>
      <th class="text-center p-4 font-semibold w-20">Actions</th>
    </tr>
  </thead>
  <tbody>
    {#if loading}
      {#each { length: 5 } as _}
        <tr class="border-t border-border" aria-hidden="true">
          <td class="px-4 py-3" colspan={5}>
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
    {:else if groups.length === 0}
      <tr class="border-t border-border">
        <td colspan={5}>
          <div class="px-4 py-16 text-center" role="status" aria-live="polite">
            <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
              <Search size={32} class="text-text-muted" />
            </div>
            <p class="text-text-primary font-semibold mt-4">
              {searchQuery ? 'No groups found' : 'No customer groups yet'}
            </p>
            <p class="text-text-muted text-sm mt-1">
              {searchQuery ? `No groups matching "${searchQuery}"` : 'Start by adding your first customer group'}
            </p>
          </div>
        </td>
      </tr>
    {:else}
      {#each groups as g}
        <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
          <td class="px-4 py-1.5 h-12 overflow-hidden">
            <div class="flex items-center gap-3">
              <div class="w-8 h-8 rounded-full bg-primary-subtle text-primary-light flex items-center justify-center text-xs font-bold shrink-0">
                {g.name?.charAt(0)?.toUpperCase() || '?'}
              </div>
              <span class="truncate font-medium">{g.name}</span>
            </div>
          </td>
          <td class="px-4 py-1.5 h-12 overflow-hidden">
            {#if g.description}
              <span class="truncate block text-text-secondary">{g.description}</span>
            {:else}
              <span class="text-text-muted">—</span>
            {/if}
          </td>
          <td class="px-4 py-1.5 h-12 overflow-hidden">
            {#if g.is_active}
              <Badge variant="success" size="sm">Active</Badge>
            {:else}
              <Badge variant="danger" size="sm">Inactive</Badge>
            {/if}
          </td>
          <td class="px-4 py-1.5 h-12 overflow-hidden text-text-secondary text-xs">
            {formatDate(g.created_at)}
          </td>
          <td class="px-4 py-1.5 h-12 overflow-hidden text-center">
            <div class="flex items-center justify-center gap-1">
              {#if canUpdate}
                <Button variant="ghost" size="icon" class="text-text-muted hover:text-primary-light transition-all active:scale-90" onclick={() => onedit(g)} title="Edit" aria-label="Edit">
                  <Pencil size={14} />
                </Button>
              {/if}
              {#if canDelete}
                <Button variant="ghost" size="icon" class="text-text-muted hover:text-danger hover:bg-danger-subtle transition-all active:scale-90" onclick={() => ondelete(g)} title="Delete" aria-label="Delete">
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
