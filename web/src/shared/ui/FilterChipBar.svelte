<script lang="ts">
  import { SlidersHorizontal, X } from 'lucide-svelte';
  import type { Component } from 'svelte';

  type Chip = {
    type: string;
    label: string;
    icon?: Component;
  };

  let {
    chips = [],
    onclear,
    onclearall,
    clearLabel = 'Clear all',
    activeClass = '',
  }: {
    chips?: Chip[];
    onclear: (type: string) => void;
    onclearall: () => void;
    clearLabel?: string;
    activeClass?: string;
  } = $props();
</script>

<div class="filter-chips-wrapper" class:is-open={chips.length > 0}>
  <div class="filter-chips-inner">
    <div class="flex flex-wrap items-center gap-2 pt-3 mt-3 border-t border-border/50">
      {#each chips as chip}
        {@const ChipIcon = chip.icon || SlidersHorizontal}
        <div class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-primary-subtle/20 border border-primary-subtle/30 rounded-full text-sm text-text-secondary {activeClass}">
          <ChipIcon size={13} class="text-primary-light shrink-0" />
          <span class="font-medium truncate max-w-[180px]">{chip.label}</span>
          <button
            class="w-4 h-4 rounded-full flex items-center justify-center text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
            onclick={() => onclear(chip.type)}
            aria-label="Clear {chip.label} filter"
          >
            <X size={12} />
          </button>
        </div>
      {/each}
      <button
        class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-text-muted hover:text-text-primary bg-surface-default/50 border border-border/50 rounded-full transition-colors"
        onclick={onclearall}
      >
        {clearLabel}
        <X size={12} />
      </button>
    </div>
  </div>
</div>

<style>
  .filter-chips-wrapper {
    display: grid;
    grid-template-rows: 0fr;
    opacity: 0;
    transition: grid-template-rows 0.2s ease-out, opacity 0.2s ease-out;
  }
  .filter-chips-wrapper.is-open {
    grid-template-rows: 1fr;
    opacity: 1;
  }
  .filter-chips-inner {
    overflow: hidden;
  }
</style>
