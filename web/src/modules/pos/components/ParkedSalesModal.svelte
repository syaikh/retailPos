<script lang="ts">
  import { fly } from 'svelte/transition';
  import { Button } from '$shared/ui';
  import { X, RotateCcw, Trash2 } from 'lucide-svelte';
  import { tick } from 'svelte';

  let dialogEl: HTMLDivElement | undefined = $state();
  let previousFocus: HTMLElement | null = null;

  let {
    showModal = $bindable(false),
    parkedSales = [],
    onrecall = (id: number) => {},
    oncancel = (id: number) => {},
  }: {
    showModal: boolean;
    parkedSales: any[];
    onrecall?: (id: number) => void;
    oncancel?: (id: number) => void;
  } = $props();

  function close() {
    showModal = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      close();
      return;
    }
  }

  $effect(() => {
    if (showModal) {
      previousFocus = document.activeElement as HTMLElement;
      tick().then(() => {
        const firstBtn = dialogEl?.querySelector<HTMLButtonElement>('button[data-action="recall"]');
        firstBtn?.focus();
      });
    } else if (previousFocus) {
      previousFocus.focus();
      previousFocus = null;
    }
  });
</script>

{#if showModal}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-[60] flex items-center justify-center" transition:fly={{ y: 40, duration: 300 }} onkeydown={handleKeydown} role="none">
    <div class="absolute inset-0 bg-black/60" onclick={close} role="presentation"></div>
    <div bind:this={dialogEl} class="relative z-[65] w-full max-w-lg max-h-[90vh] overflow-y-auto rounded-2xl border border-border-default bg-bg-card shadow-modal p-5" role="dialog" aria-modal="true" aria-labelledby="parked-modal-heading">
      <div class="flex items-center justify-between mb-4">
        <h2 id="parked-modal-heading" class="text-lg font-bold text-text-primary">Held Sales</h2>
        <button type="button" onclick={close} class="w-8 h-8 flex items-center justify-center rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover/50" aria-label="Close">
          <X size={18} />
        </button>
      </div>

      {#if parkedSales.length === 0}
        <p class="text-sm text-text-muted text-center py-8">No held sales</p>
      {:else}
        <div class="space-y-2">
          {#each parkedSales as sale}
            <div class="flex items-center justify-between px-3 py-3 rounded-lg border border-border hover:border-primary/40 transition-colors">
              <div class="min-w-0">
                <p class="text-sm font-medium text-text-primary truncate">{sale.invoice_number}</p>
                <p class="text-xs text-text-muted">
                  Rp {sale.total_amount?.toLocaleString('id-ID') || '0'} · {sale.items?.length || 0} items
                </p>
              </div>
              <div class="flex items-center gap-1.5 shrink-0 ml-3">
                <Button
                  data-action="recall"
                  onclick={() => onrecall(sale.id)}
                  variant="ghost"
                  size="sm"
                  class="text-xs text-primary-light hover:bg-primary-subtle"
                >
                  <RotateCcw size={12} />
                  Recall
                </Button>
                <Button
                  onclick={() => oncancel(sale.id)}
                  variant="ghost"
                  size="sm"
                  class="text-xs text-danger hover:bg-danger-subtle"
                >
                  <Trash2 size={12} />
                </Button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
{/if}

<style></style>
