<script lang="ts">
  import { Button, Modal } from '$shared/ui';
  import { Trash2, Users } from 'lucide-svelte';

  let {
    open = $bindable(false),
    username = '',
    deleting = false,
    subordinateCount = 0,
    oncancel = () => {},
    onconfirm = () => {},
  }: {
    open: boolean;
    username?: string;
    deleting?: boolean;
    subordinateCount?: number;
    oncancel?: () => void;
    onconfirm?: () => void;
  } = $props();
</script>

<Modal bind:open={open} title="Delete User" size="sm">
  <div class="text-center py-3">
    <div class="w-14 h-14 rounded-2xl bg-danger-subtle flex items-center justify-center mx-auto mb-4">
      <Trash2 size={24} class="text-danger" />
    </div>
    <p class="text-text-primary font-semibold mb-1">Delete "{username}"?</p>
    <p class="text-text-muted text-sm">This will permanently remove the account and all associated access.</p>
    {#if subordinateCount > 0}
      <div class="mt-4 p-3 rounded-xl bg-warning-subtle border border-warning/30 text-left">
        <div class="flex items-center gap-2 text-warning font-medium text-sm mb-1">
          <Users size={15} />
          <span>{subordinateCount} subordinate{subordinateCount !== 1 ? 's' : ''}</span>
        </div>
        <p class="text-text-secondary text-xs">This user is a manager of {subordinateCount} {subordinateCount !== 1 ? 'people' : 'person'}. Deleting will set their reports_to to null.</p>
      </div>
    {/if}
  </div>
  {#snippet footer()}
    <Button variant="secondary" onclick={oncancel}>Cancel</Button>
    <Button variant="danger" onclick={onconfirm} disabled={deleting}>Delete</Button>
  {/snippet}
</Modal>
