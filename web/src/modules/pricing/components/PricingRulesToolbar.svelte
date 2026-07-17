<script lang="ts">
  import { Button, SearchBar, Dropdown } from '$shared/ui';
  import { Plus, ChevronDown } from 'lucide-svelte';
  import { debounce } from '$shared/utils/debounce';

  let {
    searchQuery = $bindable(''),
    statusFilter = $bindable('all'),
    typeFilter = $bindable('all'),
    canCreate = false,
    pricingTypes = [],
    typeLabel = 'All Types',
    oncreate = () => {},
  }: {
    searchQuery: string;
    statusFilter: string;
    typeFilter: string;
    canCreate: boolean;
    pricingTypes: { value: string; label: string }[];
    typeLabel: string;
    oncreate?: () => void;
  } = $props();

  const debouncedSearch = debounce(() => {}, 300);

  function handleSearch() {
    debouncedSearch();
  }

  function handleFilterChange() {
    searchQuery = searchQuery;
  }
</script>

<div class="card p-4">
  <div class="flex items-center gap-3">
    <div class="flex-1">
      <SearchBar bind:value={searchQuery} placeholder="Cari rule..." oninput={handleSearch} inputClass="h-10" />
    </div>
    <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border-default" role="group" aria-label="Status filter">
      <button
        class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'all' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'all'; handleFilterChange(); }}
        aria-pressed={statusFilter === 'all'}
      >Semua</button>
      <button
        class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'active' ? 'bg-success-subtle text-success-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'active'; handleFilterChange(); }}
        aria-pressed={statusFilter === 'active'}
      >Aktif</button>
      <button
        class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'inactive' ? 'bg-danger-subtle text-danger-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'inactive'; handleFilterChange(); }}
        aria-pressed={statusFilter === 'inactive'}
      >Nonaktif</button>
    </div>
    <Dropdown placement="bottom-start" items={[
      { label: 'Semua Tipe', checked: typeFilter === 'all', onclick: () => { typeFilter = 'all'; handleFilterChange(); } },
      ...pricingTypes.map(pt => ({
        label: pt.label,
        checked: typeFilter === pt.value,
        onclick: () => { typeFilter = pt.value; handleFilterChange(); }
      }))
    ]}>
      {#snippet trigger({ toggle })}
        <button
          class="flex items-center gap-2 px-3 h-10 rounded-xl border border-border bg-surface-default text-sm hover:border-border-strong hover:bg-surface-hover transition-colors {typeFilter !== 'all' ? 'text-text-primary' : 'text-text-secondary'}"
          style="min-width: 130px;"
          onclick={toggle}
        >
          <span class="flex-1 text-left truncate">{typeLabel}</span>
          <ChevronDown size={14} class="text-text-muted shrink-0" />
        </button>
      {/snippet}
    </Dropdown>
    {#if canCreate}
      <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={oncreate}>
        <Plus size={18} /> Tambah Rule
      </Button>
    {/if}
  </div>
</div>
