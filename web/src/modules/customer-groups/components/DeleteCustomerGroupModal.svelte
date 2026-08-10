<script lang="ts">
  import { Button, Modal } from '$shared/ui';
  import { Trash2, Loader2 } from 'lucide-svelte';
  import { labels } from '$shared/i18n';

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

<Modal bind:open={open} title={labels.deleteCustomerGroup} size="sm">
  <p class="text-sm text-text-secondary">
    {labels.deleteConfirmPrefix} <strong class="text-text-primary">{targetName}</strong>? {labels.thisActionCannotBeUndone}
  </p>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={oncancel}>{labels.cancel}</Button>
    <Button variant="danger" class="px-5" disabled={deleting} onclick={onconfirm}>
      {#if deleting}
        <Loader2 size={14} class="animate-spin mr-1" /> {labels.deleting}
      {:else}
        <Trash2 size={14} class="mr-1" /> {labels.delete}
      {/if}
    </Button>
  {/snippet}
</Modal>
