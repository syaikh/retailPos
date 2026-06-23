<script lang="ts">
  import { Input } from '$shared/ui';
  import { Search, X, Loader2 } from 'lucide-svelte';

  let {
    value = $bindable(''),
    placeholder = 'Search...',
    oninput,
    loading = false,
    class: className = '',
    inputClass = '',
    id,
  }: {
    value?: string;
    placeholder?: string;
    oninput?: () => void;
    loading?: boolean;
    class?: string;
    inputClass?: string;
    id?: string;
  } = $props();

  let inputEl: HTMLInputElement;

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      if (value) {
        value = '';
        oninput?.();
      }
      inputEl?.blur();
    }
  }

  function handleClear() {
    value = '';
    oninput?.();
    inputEl?.focus();
  }
</script>

<div class="relative {className}" role="search">
  <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none shrink-0" />
  <Input
    type="text"
    {placeholder}
    bind:value
    onkeydown={handleKeydown}
    oninput={() => oninput?.()}
    class="pl-10 pr-10 w-full {inputClass}"
    bind:this={inputEl}
    {id}
    autocomplete="off"
    spellcheck="false"
  />
  {#if loading}
    <Loader2 size={14} class="absolute right-3 top-1/2 -translate-y-1/2 text-primary-light animate-spin" />
  {:else if value}
    <button
      onclick={handleClear}
      class="absolute right-3 top-1/2 -translate-y-1/2 p-0.5 text-text-muted hover:text-text-secondary transition-colors"
      aria-label="Clear search"
      type="button"
    >
      <X size={14} />
    </button>
  {/if}
</div>