<script lang="ts">
  import { Button, Modal } from '$shared/ui';
  import { Trash2 } from 'lucide-svelte';

  let {
    open = $bindable(false),
    username = '',
    deleting = false,
    oncancel = () => {},
    onconfirm = () => {},
  }: {
    open: boolean;
    username?: string;
    deleting?: boolean;
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
  </div>
  {#snippet footer()}
    <Button variant="secondary" onclick={oncancel}>Cancel</Button>
    <Button variant="danger" onclick={onconfirm} disabled={deleting}>Delete</Button>
  {/snippet}
</Modal>
