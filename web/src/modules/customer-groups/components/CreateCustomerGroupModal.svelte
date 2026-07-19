<script lang="ts">
  import { Button, Input, Modal } from '$shared/ui';
  import { UserPlus, Loader2 } from 'lucide-svelte';

  const COLORS = ['#6C5CE7', '#00B894', '#0984E3', '#E17055', '#FFD93D', '#636E72', '#E84393', '#00CEC9'];

  let {
    open = $bindable(false),
    creating = $bindable(false),
    oncreate = (data: { name: string; description?: string; color?: string }) => {},
  }: {
    open: boolean;
    creating?: boolean;
    oncreate?: (data: { name: string; description?: string; color?: string }) => void;
  } = $props();

  let name = $state('');
  let description = $state('');
  let color = $state(COLORS[0]);
  let fieldErrors = $state({ name: '' });

  $effect(() => {
    if (open) {
      name = '';
      description = '';
      color = COLORS[0];
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
      color,
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
    <div class="space-y-1" role="group" aria-labelledby="cg-color-label">
      <span id="cg-color-label" class="text-xs font-semibold text-text-secondary">Warna Avatar</span>
      <div class="flex gap-2">
        {#each COLORS as c}
          <button
            type="button"
            class="w-7 h-7 rounded-full border-2 transition-all {color === c ? 'border-white scale-110' : 'border-transparent hover:scale-105'}"
            style="background-color: {c};"
            onclick={() => color = c}
            aria-label="Pilih warna {c}"
            aria-pressed={color === c}
          ></button>
        {/each}
      </div>
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
