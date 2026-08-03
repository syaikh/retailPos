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

<Modal bind:open={open} title="Hapus Lokasi Penyimpanan" size="sm">
  <p class="text-sm text-text-secondary">
    Apakah Anda yakin ingin menghapus <strong class="text-text-primary">{targetName}</strong>? Tindakan ini tidak dapat dibatalkan.
  </p>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={oncancel}>Batal</Button>
    <Button variant="danger" class="px-5" disabled={deleting} onclick={onconfirm}>
      {#if deleting}
        <Loader2 size={14} class="animate-spin mr-1" /> Menghapus...
      {:else}
        <Trash2 size={14} class="mr-1" /> Hapus
      {/if}
    </Button>
  {/snippet}
</Modal>
