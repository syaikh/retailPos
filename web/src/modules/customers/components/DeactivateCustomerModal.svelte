<script lang="ts">
  import { Button, Modal } from '$shared/ui';
  import { Trash2, Loader2 } from 'lucide-svelte';

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

<Modal bind:open={open} title="Deactivate Customer" size="sm">
  <p class="text-sm text-text-secondary">
    Are you sure you want to deactivate <strong class="text-text-primary">{targetName}</strong>? This will hide them from active listings but preserve their history.
  </p>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={oncancel}>Cancel</Button>
    <Button variant="danger" class="px-5" disabled={deactivating} onclick={onconfirm}>
      {#if deactivating}
        <Loader2 size={14} class="animate-spin mr-1" /> Deactivating...
      {:else}
        <Trash2 size={14} class="mr-1" /> Deactivate
      {/if}
    </Button>
  {/snippet}
</Modal>
