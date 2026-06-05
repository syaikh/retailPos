<script lang="ts">
import { onMount } from 'svelte';
import { MoreVertical, Package, Pencil, Trash2 } from 'lucide-svelte';

let {
  product,
  canEdit = false,
  canDelete = false,
  onView,
  onEdit,
  onDelete,
}: {
  product: any;
  canEdit?: boolean;
  canDelete?: boolean;
  onView?: () => void;
  onEdit?: () => void;
  onDelete?: () => void;
} = $props();

let showDropdown = $state(false);
let dropdownRef = $state<HTMLDivElement | null>(null);
let buttonRef = $state<HTMLButtonElement | null>(null);

function toggleDropdown() {
  // Close other dropdowns by dispatching a custom event
  if (!showDropdown) {
    document.dispatchEvent(new CustomEvent('close-all-dropdowns'));
  }
  showDropdown = !showDropdown;
}

function closeDropdown() {
  showDropdown = false;
}

onMount(() => {
  const closeHandler = () => {
    showDropdown = false;
  };
  document.addEventListener('close-all-dropdowns', closeHandler);
  document.addEventListener('click', handleClickOutside);
  return () => {
    document.removeEventListener('close-all-dropdowns', closeHandler);
    document.removeEventListener('click', handleClickOutside);
  };
});

function handleClickOutside(e: MouseEvent) {
  if (showDropdown && dropdownRef && !dropdownRef.contains(e.target as Node) && buttonRef && !buttonRef.contains(e.target as Node)) {
    showDropdown = false;
  }
}

function handleAction(action: 'view' | 'edit' | 'delete') {
  closeDropdown();
  if (action === 'view' && onView) onView();
  if (action === 'edit' && onEdit) onEdit();
  if (action === 'delete' && onDelete) onDelete();
}
</script>

<div class="relative inline-block">
  <button
    bind:this={buttonRef}
    onclick={(e) => {
      e.stopPropagation();
      toggleDropdown();
    }}
    class="p-1.5 rounded-lg transition-colors hover:bg-surface-hover text-text-muted hover:text-text-primary"
    title="Actions"
    aria-label="Product actions"
  >
    <MoreVertical size={14} />
  </button>

  {#if showDropdown}
    <div
      bind:this={dropdownRef}
      class="absolute right-0 top-full mt-1 w-48 card-glass border border-border rounded-lg shadow-lg z-50 py-1"
      role="menu"
      aria-orientation="vertical"
      tabindex="-1"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => {
        if (e.key === 'Escape') {
          e.stopPropagation();
          closeDropdown();
        }
      }}
    >
      <button
        onclick={() => handleAction('view')}
        class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover rounded-t-lg transition-colors"
        role="menuitem"
      >
        <Package size={14} />
        View Details
      </button>
      {#if canEdit}
        <button
          onclick={() => handleAction('edit')}
          class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors"
          role="menuitem"
        >
          <Pencil size={14} />
          Edit Product
        </button>
      {/if}
      {#if canDelete && product.stock === 0}
        <button
          onclick={() => handleAction('delete')}
          class="w-full flex items-center gap-3 px-3 py-2 text-sm text-danger hover:bg-danger-subtle rounded-b-lg transition-colors"
          role="menuitem"
        >
          <Trash2 size={14} />
          Delete Product
        </button>
      {/if}
    </div>
  {/if}
</div>