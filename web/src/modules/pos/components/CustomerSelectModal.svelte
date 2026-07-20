<script lang="ts">
  import { fly } from 'svelte/transition';
  import { Button, Input } from '$shared/ui';
  import { X } from 'lucide-svelte';
  import { tick } from 'svelte';

  let dialogEl: HTMLDivElement | undefined = $state();
  let previousFocus: HTMLElement | null = null;

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

  function close() {
    showCustomerModal = false;
    customerSearch = '';
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      close();
      return;
    }
    if (e.key === 'Tab' && dialogEl) {
      const focusable = dialogEl.querySelectorAll<HTMLElement>(
        'button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])'
      );
      if (focusable.length === 0) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (e.shiftKey) {
        if (document.activeElement === first) {
          e.preventDefault();
          last.focus();
        }
      } else {
        if (document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    }
  }

  $effect(() => {
    if (showCustomerModal) {
      previousFocus = document.activeElement as HTMLElement;
      tick().then(() => {
        const input = dialogEl?.querySelector<HTMLInputElement>('input');
        input?.focus();
      });
    } else if (previousFocus) {
      previousFocus.focus();
      previousFocus = null;
    }
  });
</script>

{#if showCustomerModal}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-[60] flex items-center justify-center" transition:fly={{ y: 40, duration: 300 }} onkeydown={handleKeydown} role="none">
    <div class="absolute inset-0 bg-black/60" onclick={close} role="presentation"></div>
    <div bind:this={dialogEl} class="relative z-[65] w-full max-w-lg max-h-[90vh] overflow-y-auto rounded-2xl border border-border-default bg-bg-card shadow-modal p-5" role="dialog" aria-modal="true" aria-labelledby="customer-modal-heading">
      <div class="flex items-center justify-between mb-4">
        <h2 id="customer-modal-heading" class="text-lg font-bold text-text-primary">Pilih Customer</h2>
        <button type="button" onclick={close} class="w-8 h-8 flex items-center justify-center rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover/50" aria-label="Tutup">
          <X size={18} />
        </button>
      </div>
      <Input class="w-full mb-3" placeholder="Cari berdasarkan nama atau telepon..." bind:value={customerSearch} />
      {#if customerSearching}
        <p class="text-sm text-text-muted mb-2">Mencari...</p>
      {/if}
      <div class="max-h-80 overflow-y-auto space-y-1">
        <button type="button" class="w-full text-left px-3 py-2 rounded-lg border border-border hover:border-primary hover:bg-primary-subtle transition-colors" onclick={() => { close(); onselectcustomer(null); }}>
          <span class="text-sm font-medium">Walk-in / Umum</span>
        </button>
        {#each customerResults as c}
          <button type="button" class="w-full text-left px-3 py-2 rounded-lg border border-border hover:border-primary hover:bg-primary-subtle transition-colors" onclick={() => { close(); onselectcustomer(c.id); }}>
            <div class="text-sm font-medium">{c.name}</div>
            <div class="text-xs text-text-muted">{c.phone || 'tanpa telepon'} {c.email ? `· ${c.email}` : ''}</div>
          </button>
        {:else}
          {#if customerSearch.trim() && !customerSearching}
            <p class="text-sm text-text-muted text-center py-4">Customer tidak ditemukan</p>
          {/if}
        {/each}
      </div>
    </div>
  </div>
{/if}

<style></style>
