<script lang="ts">
  import { Button, SearchBar } from '$shared/ui';
  import { Plus } from 'lucide-svelte';

  let {
    searchQuery = $bindable(''),
    statusFilter = $bindable('all'),
    canCreate = false,
    onsearch = () => {},
    onstatuschange = () => {},
    oncreate = () => {},
  }: {
    searchQuery?: string;
    statusFilter?: string;
    canCreate?: boolean;
    onsearch?: () => void;
    onstatuschange?: () => void;
    oncreate?: () => void;
  } = $props();
</script>

<div class="card p-4 space-y-3">
  <div class="flex items-center gap-3">
    <div class="flex-1">
      <SearchBar bind:value={searchQuery} placeholder="Search by group name..." oninput={onsearch} />
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
    {#if canCreate}
      <Button onclick={oncreate} variant="primary" class="shrink-0 shadow-glow-primary-sm px-5">
        <Plus size={18} />
        Add Group
      </Button>
    {/if}
  </div>
</div>
