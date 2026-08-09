<script lang="ts">
import { onMount } from 'svelte';
import { MoreVertical, Package, Pencil, Trash2, ArrowUpDown } from 'lucide-svelte';
import { labels } from '$shared/i18n';

let {
  product,
  canEdit = false,
  canDelete = false,
  canAdjustStock = false,
  onView,
  onEdit,
  onDelete,
  onAdjustStock,
}: {
  product: any;
  canEdit?: boolean;
  canDelete?: boolean;
  canAdjustStock?: boolean;
  onView?: () => void;
  onEdit?: () => void;
  onDelete?: () => void;
  onAdjustStock?: () => void;
} = $props();

let showDropdown = $state(false);
let dropdownRef = $state<HTMLDivElement | null>(null);
let buttonRef = $state<HTMLButtonElement | null>(null);
let menuStyle = $state('');

function computePosition() {
  if (!buttonRef) return;
  const r = buttonRef.getBoundingClientRect();
  menuStyle = `position:fixed;top:${r.bottom + 4}px;right:${window.innerWidth - r.right}px`;
}

function toggleDropdown() {
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

$effect(() => {
  if (!showDropdown) return;
  computePosition();
  function reposition() { computePosition(); }
  window.addEventListener('scroll', reposition, { passive: true, capture: true });
  window.addEventListener('resize', reposition, { passive: true });
  return () => {
    window.removeEventListener('scroll', reposition, { capture: true } as EventListenerOptions);
    window.removeEventListener('resize', reposition);
  };
});

function handleClickOutside(e: MouseEvent) {
  if (showDropdown && dropdownRef && !dropdownRef.contains(e.target as Node) && buttonRef && !buttonRef.contains(e.target as Node)) {
    showDropdown = false;
  }
}

function handleAction(action: 'view' | 'edit' | 'delete' | 'adjust') {
  closeDropdown();
  if (action === 'view' && onView) onView();
  if (action === 'edit' && onEdit) onEdit();
  if (action === 'delete' && onDelete) onDelete();
  if (action === 'adjust' && onAdjustStock) onAdjustStock();
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
    title={labels.actions}
    aria-label={labels.productActions}
  >
    <MoreVertical size={14} />
  </button>

  {#if showDropdown}
    <div
      bind:this={dropdownRef}
      class="fixed z-50 w-48 card-glass border border-border rounded-lg shadow-lg py-1"
      style={menuStyle}
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
        {labels.lihatDetail}
      </button>
      {#if canAdjustStock}
        <button
          onclick={() => handleAction('adjust')}
          class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors"
          role="menuitem"
        >
          <ArrowUpDown size={14} />
          {labels.adjustStock}
        </button>
      {/if}
      {#if canEdit}
        <button
          onclick={() => handleAction('edit')}
          class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors"
          role="menuitem"
        >
          <Pencil size={14} />
          {labels.editProduk}
        </button>
      {/if}
      {#if canDelete && product.stock === 0}
        <button
          onclick={() => handleAction('delete')}
          class="w-full flex items-center gap-3 px-3 py-2 text-sm text-danger hover:bg-danger-subtle rounded-b-lg transition-colors"
          role="menuitem"
        >
          <Trash2 size={14} />
          {labels.deleteProduct}
        </button>
      {/if}
    </div>
  {/if}
</div>