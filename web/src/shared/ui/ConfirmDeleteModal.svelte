<script lang="ts">
  import { Button, Modal } from '$shared/ui';
  import { Trash2, Loader2 } from 'lucide-svelte';
  import { labels } from '$shared/i18n';

  let {
    open = $bindable(false),
    title = '',
    itemName = '',
    message = '',
    description = '',
    confirmLabel = '',
    cancelLabel = '',
    loading = false,
    onconfirm,
    oncancel,
  }: {
    open?: boolean;
    title?: string;
    itemName?: string;
    message?: string;
    description?: string;
    confirmLabel?: string;
    cancelLabel?: string;
    loading?: boolean;
    onconfirm: () => void;
    oncancel?: () => void;
  } = $props();

  function handleCancel() {
    open = false;
    oncancel?.();
  }
</script>

<Modal bind:open title={title || labels.confirmDelete} size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-danger-subtle flex items-center justify-center mx-auto mb-4">
      <Trash2 size={24} class="text-danger" />
    </div>
    {#if message}
      <p class="text-text-primary font-semibold mb-1">{message}</p>
    {:else if itemName}
      <p class="text-text-primary font-semibold mb-1">{labels.delete} "{itemName}"?</p>
    {:else}
      <p class="text-text-primary font-semibold mb-1">{labels.deleteConfirm}</p>
    {/if}
    {#if description}
      <p class="text-text-muted text-sm">{description}</p>
    {:else if !message && !itemName}
      <p class="text-text-muted text-sm">{labels.thisActionCannotBeUndone}</p>
    {/if}
  </div>
  {#snippet footer()}
    <Button variant="secondary" disabled={loading} onclick={handleCancel}>{cancelLabel || labels.cancel}</Button>
    <Button variant="danger" disabled={loading} onclick={onconfirm}>
      {#if loading}
        <Loader2 size={14} class="animate-spin mr-1" /> {confirmLabel || labels.delete}...
      {:else}
        {confirmLabel || labels.delete}
      {/if}
    </Button>
  {/snippet}
</Modal>
