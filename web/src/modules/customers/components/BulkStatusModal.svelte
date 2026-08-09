<script lang="ts">
  import { Button, Modal } from '$shared/ui';
  import { Loader2 } from 'lucide-svelte';
  import { labels, t } from '$shared/i18n';

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

<Modal bind:open={open} title={labels.bulkUpdateStatus} size="sm">
  <div class="py-2">
    <p class="text-text-primary font-semibold mb-3">
      {labels.setSelectedCustomersTo} <span class="text-primary-light">{isActive ? labels.active : labels.inactive}</span>
    </p>
    <p class="text-sm text-text-secondary mb-4">
      {t('affectedOfSelectedCount', { affected: affectedCount, selected: selectedCount })}
    </p>
    <div class="flex flex-wrap gap-2 justify-center">
      <button
        class="px-4 py-2 rounded-lg text-sm font-medium border transition-all {isActive ? 'bg-success-subtle border-success text-success-light' : 'bg-surface-default border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
        onclick={() => isActive = true}
      >{labels.activate}</button>
      <button
        class="px-4 py-2 rounded-lg text-sm font-medium border transition-all {!isActive ? 'bg-danger-subtle border-danger text-danger-light' : 'bg-surface-default border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
        onclick={() => isActive = false}
      >{labels.deactivate}</button>
    </div>
  </div>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" disabled={updating} onclick={oncancel}>{labels.cancel}</Button>
    <Button variant="primary" class="px-5" disabled={updating} onclick={onconfirm}>
      {#if updating}
        <Loader2 size={14} class="animate-spin mr-1" /> {labels.updating}
      {:else}
        {labels.update}
      {/if}
    </Button>
  {/snippet}
</Modal>
