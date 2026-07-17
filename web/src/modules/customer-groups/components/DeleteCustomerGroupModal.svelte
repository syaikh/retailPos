<script lang="ts">
  import { Button, Modal } from '$shared/ui';
  import { Trash2, Loader2 } from 'lucide-svelte';

  let {
    open = $bindable(false),
    targetName = '',
    deleting = $bindable(false),
    oncancel = () => {},
    onconfirm = () => {},
  }: {
    open: boolean;
    targetName?: string;
    deleting?: boolean;
    oncancel?: () => void;
    onconfirm?: () => void;
  } = $props();
</script>

<Modal bind:open={open} title="Delete Customer Group" size="sm">
  <p class="text-sm text-text-secondary">
    Are you sure you want to delete <strong class="text-text-primary">{targetName}</strong>? This action cannot be undone.
  </p>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={oncancel}>Cancel</Button>
    <Button variant="danger" class="px-5" disabled={deleting} onclick={onconfirm}>
      {#if deleting}
        <Loader2 size={14} class="animate-spin mr-1" /> Deleting...
      {:else}
        <Trash2 size={14} class="mr-1" /> Delete
      {/if}
    </Button>
  {/snippet}
</Modal>
