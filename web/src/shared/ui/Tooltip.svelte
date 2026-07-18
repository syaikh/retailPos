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
  let triggerEl: HTMLSpanElement | null = null;
  let tooltipStyle = $state('');

  function computePosition() {
    if (!triggerEl) return;
    const r = triggerEl.getBoundingClientRect();
    const gap = 8;
    let top = 0, left = 0;

    switch (placement) {
      case 'top':
        top = r.top - gap;
        left = r.left + r.width / 2;
        tooltipStyle = `position:fixed;bottom:auto;right:auto;top:${top}px;left:${left}px;transform:translate(-50%,-100%);`;
        break;
      case 'bottom':
        top = r.bottom + gap;
        left = r.left + r.width / 2;
        tooltipStyle = `position:fixed;bottom:auto;right:auto;top:${top}px;left:${left}px;transform:translate(-50%,0);`;
        break;
      case 'left':
        top = r.top + r.height / 2;
        left = r.left - gap;
        tooltipStyle = `position:fixed;bottom:auto;right:auto;top:${top}px;left:${left}px;transform:translate(-100%,-50%);`;
        break;
      case 'right':
        top = r.top + r.height / 2;
        left = r.right + gap;
        tooltipStyle = `position:fixed;bottom:auto;right:auto;top:${top}px;left:${left}px;transform:translate(0,-50%);`;
        break;
    }
  }

  function show() {
    if (!content) return;
    timeout = setTimeout(() => {
      computePosition();
      visible = true;
    }, delay);
  }

  function hide() {
    if (timeout) { clearTimeout(timeout); timeout = null; }
    visible = false;
  }
</script>

<span
  class="relative inline-flex"
  onmouseenter={show}
  onmouseleave={hide}
  onfocus={show}
  onblur={hide}
  role="presentation"
  bind:this={triggerEl}
>
  {@render children()}
  {#if visible && content}
    <span
      class="fixed z-50 pointer-events-none px-2.5 py-1.5 rounded-lg text-xs font-medium bg-surface-default/95 text-text-primary border border-border-default/50 shadow-lg backdrop-blur-sm whitespace-nowrap"
      style={tooltipStyle}
      role="tooltip"
    >
      {content}
    </span>
  {/if}
</span>
