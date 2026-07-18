<script lang="ts">
  import { SearchBar, Button, BulkActionDropdown } from '$shared/ui';
  import { Plus } from 'lucide-svelte';

  let {
    searchQuery = $bindable(''),
    statusFilter = $bindable('all'),
    canCreate = false,
    canExport = false,
    canImport = false,
    onsearch = () => {},
    onstatuschange = () => {},
    oncreate = () => {},
    onimport = () => {},
  }: {
    searchQuery?: string;
    statusFilter?: string;
    canCreate?: boolean;
    canExport?: boolean;
    canImport?: boolean;
    onsearch?: () => void;
    onstatuschange?: () => void;
    oncreate?: () => void;
    onimport?: () => void;
  } = $props();
</script>

<div class="card p-4">
  <div class="flex items-center gap-3">
    <div class="flex-1">
      <SearchBar bind:value={searchQuery} placeholder="Search suppliers by name or contact..." oninput={onsearch} inputClass="h-10" />
    </div>
    <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border-default" role="group" aria-label="Status filter">
      <button
        class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'all' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'all'; onstatuschange(); }}
        aria-pressed={statusFilter === 'all'}
      >All</button>
      <button
        class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'active' ? 'bg-success-subtle text-success-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'active'; onstatuschange(); }}
        aria-pressed={statusFilter === 'active'}
      >Active</button>
      <button
        class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'inactive' ? 'bg-danger-subtle text-danger-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'inactive'; onstatuschange(); }}
        aria-pressed={statusFilter === 'inactive'}
      >Inactive</button>
    </div>
    <BulkActionDropdown module="suppliers" {canExport} {canImport} onImport={onimport} />
    {#if canCreate}
      <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={oncreate}>
        <Plus size={18} /> Add Supplier
      </Button>
    {/if}
  </div>
</div>