<script lang="ts">
  import { Button, Input, Modal } from '$shared/ui';
  import { UserPlus, Loader2 } from 'lucide-svelte';

  let {
    open = $bindable(false),
    creating = $bindable(false),
    oncreate = (data: { name: string; description?: string }) => {},
  }: {
    open: boolean;
    creating?: boolean;
    oncreate?: (data: { name: string; description?: string }) => void;
  } = $props();

  let name = $state('');
  let description = $state('');
  let fieldErrors = $state({ name: '' });

  $effect(() => {
    if (open) {
      name = '';
      description = '';
      fieldErrors = { name: '' };
    }
  });

  function handleCreate() {
    const errors = { name: '' };
    let valid = true;

    if (!name.trim()) {
      errors.name = 'Name is required';
      valid = false;
    } else if (name.trim().length > 100) {
      errors.name = 'Name must be at most 100 characters';
      valid = false;
    }

    fieldErrors = errors;
    if (!valid) return;

    oncreate({
      name: name.trim(),
      description: description.trim() || undefined,
    });
  }
</script>

<Modal bind:open={open} title="Add Customer Group" size="sm">
  <div class="space-y-4">
    <div class="space-y-1">
      <label for="cg-name" class="text-xs font-semibold text-text-secondary">Group Name <span class="text-danger">*</span></label>
      <Input
        id="cg-name"
        class={fieldErrors.name ? 'border-danger' : ''}
        placeholder="e.g. VIP Customers"
        bind:value={name}
      />
      {#if fieldErrors.name}
        <p class="text-danger text-xs mt-0.5">{fieldErrors.name}</p>
      {/if}
    </div>
    <div class="space-y-1">
      <label for="cg-description" class="text-xs font-semibold text-text-secondary">Description</label>
      <Input
        tag="textarea"
        id="cg-description"
        class="min-h-[60px] resize-none"
        placeholder="Optional description for this group"
        bind:value={description}
      />
    </div>
  </div>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={() => open = false}>Cancel</Button>
    <Button variant="primary" class="px-5" disabled={creating} onclick={handleCreate}>
      {#if creating}
        <Loader2 size={14} class="animate-spin mr-1" /> Creating...
      {:else}
        <UserPlus size={14} class="mr-1" /> Create Group
      {/if}
    </Button>
  {/snippet}
</Modal>
