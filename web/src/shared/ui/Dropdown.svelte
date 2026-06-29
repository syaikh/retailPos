<script lang="ts">
  import { cn } from '$shared/utils/cn';
  import { fly } from 'svelte/transition';
  import type { ComponentType, Snippet } from 'svelte';

  type Placement = 'bottom-start' | 'bottom-end' | 'bottom' | 'top-start' | 'top-end' | 'top';

  export interface DropdownItem {
    label: string;
    icon?: ComponentType;
    iconClass?: string;
    disabled?: boolean;
    divider?: boolean;
    separator?: string;
    danger?: boolean;
    checked?: boolean;
    onclick?: () => void;
  }

  let {
    items = [],
    trigger,
    content,
    placement = 'bottom-end',
    open = $bindable(false),
    menu = true,
    menuClass = '',
  }: {
    items?: DropdownItem[];
    trigger: Snippet<{ open: boolean; toggle: () => void }>;
    content?: Snippet<{ close: () => void }>;
    placement?: Placement;
    open?: boolean;
    menu?: boolean;
    menuClass?: string;
  } = $props();

  let container: HTMLElement;
  let triggerEl: HTMLElement;
  let itemElements: HTMLElement[] = [];
  let focusedIndex = $state(-1);

  const placementClasses: Record<Placement, string> = {
    'bottom-start': 'left-0 top-full mt-1.5',
    'bottom-end': 'right-0 top-full mt-1.5',
    'bottom': 'left-1/2 -translate-x-1/2 top-full mt-1.5',
    'top-start': 'left-0 bottom-full mb-1.5',
    'top-end': 'right-0 bottom-full mb-1.5',
    'top': 'left-1/2 -translate-x-1/2 bottom-full mb-1.5',
  };

  function toggle() {
    open = !open;
  }

  function close() {
    open = false;
    focusedIndex = -1;
    triggerEl?.focus();
  }

  function handleItemClick(item: DropdownItem) {
    if (item.disabled) return;
    item.onclick?.();
    close();
  }

  function handleItemKeydown(e: KeyboardEvent, item: DropdownItem) {
    if (item.disabled) return;
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleItemClick(item);
    }
  }

  $effect(() => {
    if (!open) return;

    let rafId: number;

    function handleClickOutside(e: MouseEvent) {
      if (container && !container.contains(e.target as Node)) {
        close();
      }
    }

    function handleKeydown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        close();
        return;
      }

      if (!content && items.length > 0 && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
        e.preventDefault();
        const len = items.length;
        focusedIndex = e.key === 'ArrowDown'
          ? (focusedIndex + 1) % len
          : (focusedIndex - 1 + len) % len;
        itemElements[focusedIndex]?.focus();
      }
    }

    rafId = requestAnimationFrame(() => {
      window.addEventListener('click', handleClickOutside);
      window.addEventListener('keydown', handleKeydown);
    });

    return () => {
      cancelAnimationFrame(rafId);
      window.removeEventListener('click', handleClickOutside);
      window.removeEventListener('keydown', handleKeydown);
    };
  });

  $effect(() => {
    if (open && !content && itemElements.length > 0) {
      focusedIndex = 0;
      requestAnimationFrame(() => {
        itemElements[0]?.focus();
      });
    }
  });
</script>

<div bind:this={container} class="relative inline-block">
  <div bind:this={triggerEl}>
    {@render trigger({ open, toggle })}
  </div>

  {#if open}
    <div
      class={cn(
        'absolute z-50 bg-surface-default border border-border rounded-lg shadow-xl py-1 min-w-[160px]',
        placementClasses[placement],
        menu && 'card-glass',
        menuClass,
      )}
      role={menu ? 'menu' : undefined}
      aria-orientation={menu ? 'vertical' : undefined}
      tabindex="-1"
      transition:fly={{ y: -8, duration: 200 }}
      onclick={(e) => e.stopPropagation()}
    >
      {#if content}
        {@render content({ close })}
      {:else}
        {#each items as item, i}
          {#if item.divider}
            <div class="border-t border-border/50 my-1"></div>
          {:else if item.separator}
            <div class="px-3 py-1.5 text-xs font-semibold text-text-muted uppercase tracking-wide">{item.separator}</div>
          {:else}
            <button
              type="button"
              bind:this={itemElements[i]}
              class={cn(
                'w-full flex items-center gap-3 px-3 py-2 text-sm transition-colors',
                item.danger
                  ? 'text-danger hover:bg-danger-subtle'
                  : 'text-text-secondary hover:bg-surface-hover hover:text-text-primary',
                item.disabled && 'opacity-40 cursor-not-allowed',
                i === 0 && 'rounded-t-lg',
                i === items.length - 1 && 'rounded-b-lg',
              )}
              role={menu ? 'menuitem' : undefined}
              disabled={item.disabled}
              tabindex={item.disabled ? -1 : 0}
              onclick={() => handleItemClick(item)}
              onkeydown={(e) => handleItemKeydown(e, item)}
            >
              {#if item.checked !== undefined}
                <span class="w-4 shrink-0">
                  {#if item.checked}
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
                  {/if}
                </span>
              {/if}
              {#if item.icon}
                <svelte:component this={item.icon} size={14} class={item.iconClass || ''} />
              {/if}
              <span class="flex-1 text-left truncate">{item.label}</span>
            </button>
          {/if}
        {/each}
      {/if}
    </div>
  {/if}
</div>
