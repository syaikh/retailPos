<script lang="ts">
  import { Button } from '$shared/ui';
  import type { Snippet } from 'svelte';
  import { fly } from 'svelte/transition';
  import { X } from 'lucide-svelte';

  let {
    open = $bindable(false),
    title = '',
    width = 520,
    children,
    footer,
    ariaLabel,
    onclose,
  }: {
    open?: boolean;
    title?: string;
    width?: number;
    children?: Snippet;
    footer?: Snippet;
    ariaLabel?: string;
    onclose?: () => void;
  } = $props();

  let panelEl: HTMLDivElement = $state()!;
  let previousFocus: HTMLElement | null = null;

  const focusableSelector = 'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

  function handleClose() {
    open = false;
    onclose?.();
  }

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) handleClose();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      handleClose();
    }
  }

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

{#if open}
  <div
    class="fixed inset-0 z-50 bg-black/60"
    onclick={handleBackdropClick}
    transition:fly={{ duration: 200 }}
    aria-hidden="true"
  ></div>

  <div
    bind:this={panelEl}
    class="fixed inset-y-0 right-0 z-[55] bg-surface-default border-l border-border shadow-2xl flex flex-col transition-transform duration-300 ease-out"
    style="width: {width}px; max-width: 100%;"
    transition:fly={{ x: width, duration: 300, easing: t => t * (2 - t) }}
    role="dialog"
    aria-modal="true"
    aria-label={ariaLabel || title || 'Drawer'}
    tabindex="-1"
    onkeydown={(e) => { handleKeydown(e); trapFocus(e); }}
  >
    {#if title}
      <div class="flex items-center justify-between px-6 py-5 border-b border-border shrink-0">
        <h2 id="drawer-heading" class="text-base font-semibold text-text-primary">{title}</h2>
        <Button variant="ghost" size="icon" class="text-text-muted hover:text-text-primary" onclick={handleClose} aria-label="Close drawer">
          <X size={18} />
        </Button>
      </div>
    {:else}
      <Button variant="ghost" size="icon" class="text-text-muted hover:text-text-primary absolute top-4 right-4" onclick={handleClose} aria-label="Close drawer">
        <X size={18} />
      </Button>
    {/if}

    <div class="flex-1 overflow-y-auto px-6 py-4">
      {@render children?.()}
    </div>

    {#if footer}
      <div class="px-6 py-4 border-t border-border shrink-0" role="none" onkeydown={trapFocus}>
        {@render footer()}
      </div>
    {/if}
  </div>
{/if}
