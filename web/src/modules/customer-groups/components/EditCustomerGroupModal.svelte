<script lang="ts">
  import { Button, Input, Modal } from '$shared/ui';
  import { Save, Loader2 } from 'lucide-svelte';

  const COLORS = ['#6C5CE7', '#00B894', '#0984E3', '#E17055', '#FFD93D', '#636E72', '#E84393', '#00CEC9'];

  let {
    open = $bindable(false),
    group = $bindable(null as any),
    saving = $bindable(false),
    onsave = (data: any) => {},
    oncancel = () => {},
  }: {
    open: boolean;
    group: any;
    saving: boolean;
    onsave?: (data: any) => void;
    oncancel?: () => void;
  } = $props();

  let name = $state('');
  let description = $state('');
  let isActive = $state(true);
  let color = $state(COLORS[0]);
  let fieldErrors = $state({ name: '' });

  let origName = $state('');
  let origDescription = $state<string | undefined>();
  let origColor = $state(COLORS[0]);

  $effect(() => {
    if (open && group) {
      name = group.name || '';
      description = group.description || '';
      isActive = group.is_active !== false;
      color = group.color || COLORS[0];
      origName = group.name || '';
      origDescription = group.description;
      origColor = color;
      fieldErrors = { name: '' };
    }
  });

  function handleSave() {
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

    const payload: Record<string, any> = { id: group.id };
    if (name.trim() !== origName) payload.name = name.trim();
    if ((description.trim() || undefined) !== (origDescription || undefined)) payload.description = description.trim() || null;
    if (isActive !== group.is_active) payload.is_active = isActive;
    if (color !== origColor) payload.color = color;
    onsave(payload);
  }

  function handleCancel() {
    fieldErrors = { name: '' };
    oncancel();
  }
</script>

<Modal bind:open={open} title="Edit Customer Group" size="sm">
  <div class="space-y-4">
    <div class="space-y-1">
      <label for="edit-cg-name" class="text-xs font-semibold text-text-secondary">Group Name <span class="text-danger">*</span></label>
      <Input
        id="edit-cg-name"
        class={fieldErrors.name ? 'border-danger' : ''}
        placeholder="e.g. VIP Customers"
        bind:value={name}
      />
      {#if fieldErrors.name}
        <p class="text-danger text-xs mt-0.5">{fieldErrors.name}</p>
      {/if}
    </div>
    <div class="space-y-1">
      <label for="edit-cg-desc" class="text-xs font-semibold text-text-secondary">Description</label>
      <Input
        tag="textarea"
        id="edit-cg-desc"
        class="min-h-[60px] resize-none"
        placeholder="Optional description"
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
    <label class="flex items-center gap-2 text-sm">
      <input type="checkbox" bind:checked={isActive} />
      <span class="text-text-secondary font-medium">Active</span>
    </label>
  </div>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={handleCancel}>Cancel</Button>
    <Button variant="primary" class="px-5" disabled={saving} onclick={handleSave}>
      {#if saving}
        <Loader2 size={14} class="animate-spin mr-1" /> Saving...
      {:else}
        <Save size={14} class="mr-1" /> Save Changes
      {/if}
    </Button>
  {/snippet}
</Modal>
