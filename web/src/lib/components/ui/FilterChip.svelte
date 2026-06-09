<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    active = false,
    onclick = () => {},
    disabled = false,
    children,
    class: className = '',
  }: {
    active?: boolean;
    onclick?: () => void;
    disabled?: boolean;
    children?: Snippet;
    class?: string;
  } = $props();

  function handleKeydown(e: KeyboardEvent) {
    if (disabled) return;
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      onclick();
    }
  }

  function handleClick(e: MouseEvent) {
    if (disabled) {
      e.preventDefault();
      e.stopPropagation();
      return;
    }
    onclick();
  }
</script>

<button
  type="button"
  disabled={disabled}
  onclick={handleClick}
  onkeydown={handleKeydown}
  class="flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium transition-all focus:outline-none focus:ring-2 focus:ring-primary-light focus:ring-offset-1 focus:ring-offset-bg-secondary {active ? 'bg-primary-default text-white border border-primary-default shadow-sm' : 'bg-surface-default text-text-secondary border border-border hover:border-border-strong hover:text-text-primary hover:bg-surface-hover'} {disabled ? 'opacity-40 cursor-not-allowed hover:bg-surface-default hover:text-text-secondary hover:border-border' : ''} {className}"
>
  {@render children?.()}
</button>
