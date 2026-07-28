<script lang="ts">
  import { SearchBar, Button, Input } from '$shared/ui';
  import { Plus } from 'lucide-svelte';

  let {
    searchQuery = $bindable(''),
    statusFilter = $bindable(''),
    startDate = $bindable(''),
    endDate = $bindable(''),
    canCreate = false,
    onsearch = () => {},
    onstatuschange = () => {},
    onstartdatechange = () => {},
    onenddatechange = () => {},
    oncreate = () => {},
  }: {
    searchQuery?: string;
    statusFilter?: string;
    startDate?: string;
    endDate?: string;
    canCreate?: boolean;
    onsearch?: () => void;
    onstatuschange?: () => void;
    onstartdatechange?: () => void;
    onenddatechange?: () => void;
    oncreate?: () => void;
  } = $props();
</script>

<div class="card p-4">
  <div class="flex flex-wrap items-center gap-3">
    <div class="flex-1 min-w-[200px]">
      <SearchBar bind:value={searchQuery} placeholder="Search PO number or supplier..." oninput={onsearch} inputClass="h-9" />
    </div>
    <div class="flex items-center gap-2">
      <Input tag="select" bind:value={statusFilter} onchange={onstatuschange} class="h-9 w-[160px]">
        <option value="">All Status</option>
        <option value="draft">Draft</option>
        <option value="confirmed">Confirmed</option>
        <option value="partial_received">Partial Received</option>
        <option value="fully_received">Fully Received</option>
        <option value="cancelled">Cancelled</option>
      </Input>
      <Input type="date" bind:value={startDate} oninput={onstartdatechange} class="h-9 w-[160px]" />
      <Input type="date" bind:value={endDate} oninput={onenddatechange} class="h-9 w-[160px]" />
    </div>
    {#if canCreate}
      <Button variant="primary" class="shrink-0 shadow-glow-primary-sm" onclick={oncreate}>
        <Plus size={18} /> Create PO
      </Button>
    {/if}
  </div>
</div>
