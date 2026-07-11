<script lang="ts">
  import { Button, Modal } from '$shared/ui';
  import { Trash2, Loader2 } from 'lucide-svelte';

  let {
    open = $bindable(false),
    title = 'Confirm Delete',
    itemName = '',
    message = '',
    description = '',
    confirmLabel = 'Delete',
    cancelLabel = 'Cancel',
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

<Modal bind:open {title} size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-danger-subtle flex items-center justify-center mx-auto mb-4">
      <Trash2 size={24} class="text-danger" />
    </div>
    {#if message}
      <p class="text-text-primary font-semibold mb-1">{message}</p>
    {:else if itemName}
      <p class="text-text-primary font-semibold mb-1">{title.includes('Hapus') ? 'Hapus' : 'Delete'} "{itemName}"?</p>
    {:else}
      <p class="text-text-primary font-semibold mb-1">Are you sure you want to delete this item?</p>
    {/if}
    {#if description}
      <p class="text-text-muted text-sm">{description}</p>
    {:else if !message && !itemName}
      <p class="text-text-muted text-sm">This action cannot be undone.</p>
    {/if}
  </div>
  {#snippet footer()}
    <Button variant="secondary" disabled={loading} onclick={handleCancel}>{cancelLabel}</Button>
    <Button variant="danger" disabled={loading} onclick={onconfirm}>
      {#if loading}
        <Loader2 size={14} class="animate-spin mr-1" /> {confirmLabel}...
      {:else}
        {confirmLabel}
      {/if}
    </Button>
  {/snippet}
</Modal>
