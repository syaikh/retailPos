<script lang="ts">
  import { fly } from 'svelte/transition';
  import { Button, Input } from '$shared/ui';
  import { X } from 'lucide-svelte';

  let {
    showCustomerModal = $bindable(false),
    customerSearch = $bindable(''),
    customerResults = [],
    customerSearching = false,
    onselectcustomer = (id: number | null) => {},
  }: {
    showCustomerModal: boolean;
    customerSearch: string;
    customerResults: any[];
    customerSearching: boolean;
    onselectcustomer?: (id: number | null) => void;
  } = $props();
</script>

{#if showCustomerModal}
  <div class="fixed inset-0 z-[60] flex items-center justify-center" transition:fly={{ y: 40, duration: 300 }}>
    <div class="absolute inset-0 bg-black/60" onclick={() => (showCustomerModal = false)} role="presentation"></div>
    <div class="relative z-[65] w-full max-w-lg rounded-2xl border border-border-default bg-bg-card shadow-modal p-5" role="dialog" aria-modal="true" aria-labelledby="customer-modal-heading">
      <div class="flex items-center justify-between mb-4">
        <h2 id="customer-modal-heading" class="text-lg font-bold text-text-primary">Select Customer</h2>
        <button type="button" onclick={() => (showCustomerModal = false)} class="w-8 h-8 flex items-center justify-center rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover/50" aria-label="Close customer selection">
          <X size={18} />
        </button>
      </div>
      <Input class="w-full mb-3" placeholder="Search by phone or name..." bind:value={customerSearch} />
      {#if customerSearching}
        <p class="text-sm text-text-muted mb-2">Searching...</p>
      {/if}
      <div class="max-h-80 overflow-y-auto space-y-1">
        <button type="button" class="w-full text-left px-3 py-2 rounded-lg border border-border hover:border-primary hover:bg-primary-subtle transition-colors" onclick={() => { showCustomerModal = false; onselectcustomer(null); }}>
          <span class="text-sm font-medium">Walk-in / General</span>
        </button>
        {#each customerResults as c}
          <button type="button" class="w-full text-left px-3 py-2 rounded-lg border border-border hover:border-primary hover:bg-primary-subtle transition-colors" onclick={() => { showCustomerModal = false; onselectcustomer(c.id); customerSearch = ''; }}>
            <div class="text-sm font-medium">{c.name}</div>
            <div class="text-xs text-text-muted">{c.phone || 'no phone'} {c.email ? `· ${c.email}` : ''}</div>
          </button>
        {:else}
          {#if customerSearch.trim() && !customerSearching}
            <p class="text-sm text-text-muted text-center py-4">No customers found</p>
          {/if}
        {/each}
      </div>
    </div>
  </div>
{/if}

<style></style>
