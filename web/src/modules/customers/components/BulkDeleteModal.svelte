<script lang="ts">
  import { Button, Modal } from '$shared/ui';
  import { Trash2, Loader2 } from 'lucide-svelte';

  let {
    open = $bindable(false),
    count = 0,
    deleting = $bindable(false),
    oncancel = () => {},
    onconfirm = () => {},
  }: {
    open: boolean;
    count?: number;
    deleting?: boolean;
    oncancel?: () => void;
    onconfirm?: () => void;
  } = $props();
</script>

<Modal bind:open={open} title="Delete Customers" size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-danger-subtle flex items-center justify-center mx-auto mb-4">
      <Trash2 size={24} class="text-danger" />
    </div>
    <p class="text-text-primary font-semibold mb-1">Delete {count} customer(s)?</p>
    <p class="text-text-muted text-sm">This will permanently remove them from the active customer list. Their transaction history will be preserved.</p>
  </div>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" disabled={deleting} onclick={oncancel}>Cancel</Button>
    <Button variant="danger" class="px-5" disabled={deleting} onclick={onconfirm}>
      {#if deleting}
        <Loader2 size={14} class="animate-spin mr-1" /> Deleting...
      {:else}
        Delete
      {/if}
    </Button>
  {/snippet}
</Modal>
