<script lang="ts">
  import { Button, Modal } from '$shared/ui';
  import { Trash2, Loader2 } from 'lucide-svelte';
  import { labels } from '$shared/i18n';

  let {
    open = $bindable(false),
    targetName = '',
    deactivating = $bindable(false),
    oncancel = () => {},
    onconfirm = () => {},
  }: {
    open: boolean;
    targetName?: string;
    deactivating?: boolean;
    oncancel?: () => void;
    onconfirm?: () => void;
  } = $props();
</script>

<Modal bind:open={open} title={labels.deactivateCustomer} size="sm">
  <p class="text-sm text-text-secondary">
    {labels.deactivateConfirmPrefix} <strong class="text-text-primary">{targetName}</strong>? {labels.deactivateConfirmSuffix}
  </p>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={oncancel}>{labels.cancel}</Button>
    <Button variant="danger" class="px-5" disabled={deactivating} onclick={onconfirm}>
      {#if deactivating}
        <Loader2 size={14} class="animate-spin mr-1" /> {labels.deactivating}
      {:else}
        <Trash2 size={14} class="mr-1" /> {labels.deactivate}
      {/if}
    </Button>
  {/snippet}
</Modal>
