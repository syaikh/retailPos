<script lang="ts">
  import { SearchBar, Button, Dropdown } from '$shared/ui';
  import { Plus, ChevronDown } from 'lucide-svelte';
  import { STOCK_OPNAME_STATUS_LABELS } from '../types';

  let {
    searchQuery = $bindable(''),
    statusFilter = $bindable(''),
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

  const statusOptions = Object.entries(STOCK_OPNAME_STATUS_LABELS).map(([value, label]) => ({ value, label }));

  const statusLabel = $derived(
    statusOptions.find(s => s.value === statusFilter)?.label || 'All Status'
  );

  const statusItems = $derived([
    { label: 'All Status', checked: statusFilter === '', onclick: () => { statusFilter = ''; onstatuschange(); } },
    ...statusOptions.map(opt => ({
      label: opt.label,
      checked: statusFilter === opt.value,
      onclick: () => { statusFilter = opt.value; onstatuschange(); },
    })),
  ]);
</script>

<div class="card p-3">
  <div class="flex flex-wrap items-center gap-3">
    <div class="min-w-0 flex-[2_1_200px]">
      <SearchBar bind:value={searchQuery} placeholder="Search session number..." oninput={onsearch} inputClass="h-10" />
    </div>
    <Dropdown placement="bottom-start" items={statusItems}>
      {#snippet trigger({ toggle })}
        <button
          type="button"
          class="flex items-center gap-2 px-3 h-10 rounded-xl border transition-all duration-200 text-[13px] font-medium whitespace-nowrap {statusFilter !== '' ? 'bg-primary/10 border-primary/30 text-primary-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
          onclick={toggle}
        >
          <span>{statusLabel}</span>
          <ChevronDown size={14} class="text-text-muted shrink-0" />
        </button>
      {/snippet}
    </Dropdown>
    {#if canCreate}
      <Button variant="primary" class="shrink-0 shadow-glow-primary-sm" onclick={oncreate}>
        <Plus size={18} /> New Stock Opname
      </Button>
    {/if}
  </div>
</div>
