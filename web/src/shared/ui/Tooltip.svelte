<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    content = '',
    placement = 'top',
    delay = 300,
    children,
  }: {
    content: string;
    placement?: 'top' | 'bottom' | 'left' | 'right';
    delay?: number;
    children: Snippet;
  } = $props();

  let visible = $state(false);
  let timeout: ReturnType<typeof setTimeout> | null = null;

  function show() {
    if (!content) return;
    timeout = setTimeout(() => { visible = true; }, delay);
  }

  function hide() {
    if (timeout) { clearTimeout(timeout); timeout = null; }
    visible = false;
  }

  const placementClasses: Record<string, string> = {
    top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
    bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
    left: 'right-full top-1/2 -translate-y-1/2 mr-2',
    right: 'left-full top-1/2 -translate-y-1/2 ml-2',
  };
</script>

<span
  class="relative inline-flex"
  onmouseenter={show}
  onmouseleave={hide}
  onfocus={show}
  onblur={hide}
  role="presentation"
>
  {@render children()}
  {#if visible && content}
    <span
      class="absolute z-50 pointer-events-none px-2.5 py-1.5 rounded-lg text-xs font-medium bg-surface-default/95 text-text-primary border border-border-default/50 shadow-lg backdrop-blur-sm whitespace-nowrap {placementClasses[placement]}"
      role="tooltip"
    >
      {content}
    </span>
  {/if}
</span>
