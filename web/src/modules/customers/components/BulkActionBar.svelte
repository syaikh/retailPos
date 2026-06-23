<script lang="ts">
  import { Button } from '$shared/ui';

  let {
    selectedCount = 0,
    canUpdate = false,
    canDelete = false,
    onstatus = () => {},
    ondelete = () => {},
    onclear = () => {},
  }: {
    selectedCount?: number;
    canUpdate?: boolean;
    canDelete?: boolean;
    onstatus?: () => void;
    ondelete?: () => void;
    onclear?: () => void;
  } = $props();
</script>

{#if selectedCount > 0}
  <div class="px-4 py-2.5 bg-primary/5 border-t border-primary/20 flex items-center gap-3">
    <span class="text-sm font-semibold text-text-primary">{selectedCount} selected</span>
    <div class="flex items-center gap-2 ml-auto">
      {#if canUpdate}
        <Button variant="secondary" class="text-xs px-3 py-1.5 h-auto" onclick={onstatus}>
          Change Status
        </Button>
      {/if}
      {#if canDelete}
        <Button variant="danger" class="text-xs px-3 py-1.5 h-auto" onclick={ondelete}>Delete</Button>
      {/if}
      <button type="button" class="text-xs px-3 py-1.5 h-auto text-text-muted hover:text-text-secondary transition-colors font-medium" onclick={onclear}>Clear</button>
    </div>
  </div>
{/if}
