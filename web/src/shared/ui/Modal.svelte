<script lang="ts">
  import { Button } from '$shared/ui';
  import type { Snippet } from 'svelte';
  import { X } from 'lucide-svelte';
  import { fade, fly } from 'svelte/transition';

  let {
    open = $bindable(false),
    title = '',
    size = 'md',
    persistent = false,
    panelClass = '',
    children,
    footer,
  }: {
    open?: boolean;
    title?: string;
    size?: 'sm' | 'md' | 'lg' | 'xl';
    persistent?: boolean;
    panelClass?: string;
    children: Snippet;
    footer?: Snippet;
  } = $props();

  let panelEl: HTMLDivElement = $state()!;
  let previousFocus: HTMLElement | null = null;

  const sizes = {
    sm: 'max-w-sm',
    md: 'max-w-lg',
    lg: 'max-w-2xl',
    xl: 'max-w-4xl',
  };

  const focusableSelector = 'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

  function trapFocus(e: KeyboardEvent) {
    if (e.key !== 'Tab' || !panelEl) return;
    const focusable = panelEl.querySelectorAll<HTMLElement>(focusableSelector);
    if (focusable.length === 0) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (e.shiftKey) {
      if (document.activeElement === first) {
        e.preventDefault();
        last.focus();
      }
    } else {
      if (document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && !persistent) open = false;
  }

  function stopPropagation(e: MouseEvent) {
    e.stopPropagation();
  }

  function stopPropagationKey(e: KeyboardEvent) {
    e.stopPropagation();
  }

  $effect(() => {
    if (open) {
      previousFocus = document.activeElement as HTMLElement;
      requestAnimationFrame(() => {
        if (panelEl) {
          const focusable = panelEl.querySelector<HTMLElement>(focusableSelector);
          if (focusable) focusable.focus();
          else panelEl.focus();
        }
      });
    } else if (previousFocus) {
      previousFocus.focus();
      previousFocus = null;
    }
  });
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <!-- Backdrop -->
  <div
    class="fixed inset-0 z-[70] flex items-center justify-center p-4 bg-black/60"
    transition:fade={{ duration: 200 }}
    role="presentation"
    onclick={() => { if (!persistent) open = false; }}
    onkeydown={(e) => { if (e.key === 'Escape' && !persistent) open = false; }}
  >
    <!-- Panel - trap focus within the dialog -->
    <div
      bind:this={panelEl}
      class="relative w-full {sizes[size]} bg-surface-default border border-border rounded-2xl shadow-modal max-h-[85vh] flex flex-col {panelClass}"
      transition:fly={{ y: 20, duration: 300 }}
      role="dialog"
      aria-modal="true"
      aria-label={title || 'Dialog'}
      tabindex="-1"
      onclick={stopPropagation}
      onkeydown={trapFocus}
    >
      <!-- Header -->
      {#if title}
        <div class="flex items-center justify-between px-6 py-4 border-b border-border">
          <h2 class="text-base font-semibold text-text-primary">{title}</h2>
          <Button variant="ghost" size="icon" class="text-text-muted hover:text-text-primary" onclick={() => open = false} aria-label="Close modal">
            <X size={18} />
          </Button>
        </div>
      {:else}
        <Button variant="ghost" size="icon" class="text-text-muted hover:text-text-primary absolute top-4 right-4" onclick={() => open = false} aria-label="Close modal">
          <X size={18} />
        </Button>
      {/if}

      <!-- Body -->
      <div class="flex-1 overflow-y-auto">
        <div class="px-6 py-5">
          {@render children()}
        </div>
      </div>

      <!-- Footer -->
      {#if footer}
        <div class="px-6 py-4 border-t border-border flex items-center justify-end gap-3" role="none" onkeydown={trapFocus}>
          {@render footer()}
        </div>
      {/if}
    </div>
  </div>
{/if}