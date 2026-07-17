<script lang="ts">
  import { Button, SearchBar, BulkActionDropdown } from '$shared/ui';
  import { Plus } from 'lucide-svelte';

  let {
    searchQuery = $bindable(''),
    statusFilter = $bindable('all'),
    groupFilter = $bindable('all'),
    canCreate = false,
    groups = [] as { id: number; name: string }[],
    onsearch = () => {},
    onstatuschange = () => {},
    ongroupchange = () => {},
    oncreate = () => {},
    onImport = () => {},
  }: {
    searchQuery?: string;
    statusFilter?: string;
    groupFilter?: string;
    canCreate?: boolean;
    groups?: { id: number; name: string }[];
    onsearch?: () => void;
    onstatuschange?: () => void;
    ongroupchange?: () => void;
    oncreate?: () => void;
    onImport?: () => void;
  } = $props();
</script>

<div class="card p-4 space-y-3">
  <div class="flex items-center gap-3">
    <div class="flex-1">
      <SearchBar bind:value={searchQuery} placeholder="Search by name, phone, or email..." oninput={onsearch} />
    </div>
    <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border-default">
      <button
        class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'all' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'all'; onstatuschange(); }}
      >
        All
      </button>
      <button
        class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'active' ? 'bg-success-subtle text-success-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'active'; onstatuschange(); }}
      >
        Active
      </button>
      <button
        class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'inactive' ? 'bg-danger-subtle text-danger-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'inactive'; onstatuschange(); }}
      >
        Inactive
      </button>
    </div>
    <select
      class="h-8 px-3 rounded-lg border border-border-default bg-bg-secondary text-xs font-medium text-text-secondary hover:border-border-strong hover:bg-surface-hover transition-colors"
      bind:value={groupFilter}
      onchange={() => ongroupchange()}
    >
      <option value="all">All Groups</option>
      {#each groups as g}
        <option value={String(g.id)}>{g.name}</option>
      {/each}
    </select>
    {#if canCreate}
      <BulkActionDropdown module="customers" canExport={canCreate} canImport={canCreate} {onImport} />
      <Button onclick={oncreate} variant="primary" class="shrink-0 shadow-glow-primary-sm px-5">
        <Plus size={18} />
        Add Customer
      </Button>
    {/if}
  </div>
</div>
