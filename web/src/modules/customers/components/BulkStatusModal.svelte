<script lang="ts">
  import { Button, Modal } from '$shared/ui';
  import { Loader2 } from 'lucide-svelte';

  let {
    open = $bindable(false),
    selectedCount = 0,
    affectedCount = 0,
    isActive = $bindable(true),
    updating = $bindable(false),
    oncancel = () => {},
    onconfirm = () => {},
  }: {
    open: boolean;
    selectedCount?: number;
    affectedCount?: number;
    isActive?: boolean;
    updating?: boolean;
    oncancel?: () => void;
    onconfirm?: () => void;
  } = $props();
</script>

<Modal bind:open={open} title="Bulk Update Status" size="sm">
  <div class="py-2">
    <p class="text-text-primary font-semibold mb-3">
      Set selected customers to <span class="text-primary-light">{isActive ? 'Active' : 'Inactive'}</span>
    </p>
    <p class="text-sm text-text-secondary mb-4">
      {affectedCount} of {selectedCount} customer(s) will be updated.
    </p>
    <div class="flex flex-wrap gap-2 justify-center">
      <button
        class="px-4 py-2 rounded-lg text-sm font-medium border transition-all {isActive ? 'bg-success-subtle border-success text-success-light' : 'bg-surface-default border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
        onclick={() => isActive = true}
      >Activate</button>
      <button
        class="px-4 py-2 rounded-lg text-sm font-medium border transition-all {!isActive ? 'bg-danger-subtle border-danger text-danger-light' : 'bg-surface-default border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
        onclick={() => isActive = false}
      >Deactivate</button>
    </div>
  </div>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" disabled={updating} onclick={oncancel}>Cancel</Button>
    <Button variant="primary" class="px-5" disabled={updating} onclick={onconfirm}>
      {#if updating}
        <Loader2 size={14} class="animate-spin mr-1" /> Updating...
      {:else}
        Update
      {/if}
    </Button>
  {/snippet}
</Modal>
