<script lang="ts">
  import { cn } from '$shared/utils/cn';
  import { Search, ChevronDown } from 'lucide-svelte';

  interface Option {
    value: number;
    label: string;
  }

  let {
    value = $bindable(),
    options = [] as Option[],
    placeholder = 'Select...',
    searchPlaceholder = 'Search...',
    disabled = false,
    class: className = '',
    notFoundText = 'No results found',
    onchange,
  }: {
    value?: number;
    options?: Option[];
    placeholder?: string;
    searchPlaceholder?: string;
    disabled?: boolean;
    class?: string;
    notFoundText?: string;
    onchange?: (value: number) => void;
  } = $props();

  let open = $state(false);
  let search = $state('');
  let container: HTMLDivElement;
  let searchInput: HTMLInputElement;
  let buttonEl: HTMLButtonElement;

  const selectedLabel = $derived(options.find(o => o.value === value)?.label || '');

  const filteredOptions = $derived(
    search ? options.filter(o => o.label.toLowerCase().includes(search.toLowerCase())) : options
  );

  function openDropdown() {
    if (disabled) return;
    open = true;
    search = '';
  }

  function handleSelect(opt: Option) {
    value = opt.value;
    open = false;
    search = '';
    onchange?.(opt.value);
  }

  $effect(() => {
    if (open) {
      requestAnimationFrame(() => searchInput?.focus());
    }
  });

  $effect(() => {
    if (!open) return;

    function handleClickOutside(e: MouseEvent) {
      if (container && !container.contains(e.target as Node)) {
        open = false;
        search = '';
      }
    }

    function handleKeydown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        open = false;
        search = '';
      }
    }

    const raf = requestAnimationFrame(() => {
      window.addEventListener('click', handleClickOutside);
      window.addEventListener('keydown', handleKeydown);
    });

    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener('click', handleClickOutside);
      window.removeEventListener('keydown', handleKeydown);
    };
  });
</script>

<div bind:this={container} class={cn('relative', className)}>
  <button
    bind:this={buttonEl}
    type="button"
    onclick={openDropdown}
    disabled={disabled}
    class={cn(
      'w-full rounded-xl border bg-bg-secondary px-3.5 py-2.5 text-sm text-left transition-colors duration-200 flex items-center gap-2',
      open ? 'border-primary-default ring-2 ring-primary/30' : 'border-border-default',
      disabled && 'opacity-40 cursor-not-allowed',
      'hover:border-primary-default focus:outline-none focus:ring-2 focus:ring-primary/30',
      !selectedLabel && 'text-text-muted'
    )}
    aria-haspopup="listbox"
    aria-expanded={open}
  >
    <span class="flex-1 truncate">{selectedLabel || placeholder}</span>
    <ChevronDown size={16} class={cn('text-text-muted transition-transform duration-200 shrink-0', open && 'rotate-180')} />
  </button>

  {#if open}
    <div
      style="position: absolute; top: calc(100% + 6px); left: 0; width: 100%;"
      class="z-[60] bg-surface-default border border-border rounded-xl shadow-xl py-1 overflow-hidden"
      role="listbox"
    >
      <div class="px-2 pb-1.5">
        <div class="flex items-center gap-2 bg-bg-secondary border border-border-default rounded-lg px-3 py-1.5 text-sm">
          <Search size={14} class="text-text-muted shrink-0" />
          <input
            bind:this={searchInput}
            type="text"
            bind:value={search}
            placeholder={searchPlaceholder}
            class="w-full bg-transparent outline-none text-text-primary placeholder:text-text-muted"
          />
        </div>
      </div>
      <div class="overflow-y-auto max-h-52">
        {#if filteredOptions.length === 0}
          <div class="px-3 py-4 text-sm text-text-muted text-center">{notFoundText}</div>
        {:else}
          {#each filteredOptions as opt}
            <button
              type="button"
              onclick={() => handleSelect(opt)}
              class={cn(
                'w-full text-left px-3 py-2 text-sm transition-colors',
                opt.value === value
                  ? 'bg-primary/10 text-primary-default font-medium'
                  : 'text-text-secondary hover:bg-surface-hover hover:text-text-primary'
              )}
              role="option"
              aria-selected={opt.value === value}
            >
              {opt.label}
            </button>
          {/each}
        {/if}
      </div>
    </div>
  {/if}
</div>
