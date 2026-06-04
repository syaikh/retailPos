<script lang="ts">
  import type { Snippet } from 'svelte';
  import { X } from 'lucide-svelte';
  import { fade, fly } from 'svelte/transition';

  let {
    open = $bindable(false),
    title = '',
    size = 'md',
    children,
    footer,
  }: {
    open?: boolean;
    title?: string;
    size?: 'sm' | 'md' | 'lg' | 'xl';
    children: Snippet;
    footer?: Snippet;
  } = $props();

  const sizes = {
    sm: 'max-w-sm',
    md: 'max-w-lg',
    lg: 'max-w-2xl',
    xl: 'max-w-5xl',
  };

  // Only allow Escape key to close - remove backdrop click handler
  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') open = false;
  }

  function stopPropagation(e: MouseEvent) {
    e.stopPropagation();
  }

  function stopPropagationKey(e: KeyboardEvent) {
    e.stopPropagation();
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <!-- Backdrop - no click handler to prevent closing when clicking outside -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm"
    transition:fade={{ duration: 200 }}
  >
    <!-- Panel - stop propagation to prevent events bubbling to backdrop -->
    <div
      class="relative w-full {sizes[size]} bg-surface border border-border rounded-2xl shadow-modal max-h-[85vh] flex flex-col"
      transition:fly={{ y: 20, duration: 300 }}
      role="dialog"
      aria-modal="true"
      aria-label={title || 'Dialog'}
      tabindex="-1"
      onclick={stopPropagation}
      onkeydown={stopPropagationKey}
    >
      <!-- Header -->
      {#if title}
        <div class="flex items-center justify-between px-6 py-4 border-b border-border">
          <h2 class="text-base font-semibold text-text-primary">{title}</h2>
          <button
            onclick={() => open = false}
            class="btn-icon btn-ghost text-text-muted hover:text-text-primary"
            aria-label="Close modal"
          >
            <X size={18} />
          </button>
        </div>
      {:else}
        <button
          onclick={() => open = false}
          class="btn-icon btn-ghost text-text-muted hover:text-text-primary absolute top-4 right-4"
          aria-label="Close modal"
        >
          <X size={18} />
        </button>
      {/if}

      <!-- Body -->
      <div class="flex-1 overflow-y-auto">
        <div class="px-6 py-5">
          {@render children()}
        </div>
      </div>

      <!-- Footer -->
      {#if footer}
        <div class="px-6 py-4 border-t border-border flex items-center justify-end gap-3">
          {@render footer()}
        </div>
      {/if}
    </div>
  </div>
{/if}