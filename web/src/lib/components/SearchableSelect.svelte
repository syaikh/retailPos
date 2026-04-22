<script lang="ts">
  import { Search, ChevronDown, Check } from 'lucide-svelte';

  /** 
   * @typedef {{ value: any; label: string }} Option
   */

  let { 
    options = [], 
    value = $bindable<any>(), 
    placeholder = "Pilih opsi...",
    width = "250px"
  } = $props<{
    options: { value: any; label: string }[];
    value: any;
    placeholder?: string;
    width?: string;
  }>();

  let isOpen = $state(false);
  let searchQuery = $state('');

  let filteredOptions = $derived(
    options.filter((o: { value: any; label: string }) => {
      if (o.value === 'all') return true;
      return o.label.toLowerCase().includes(searchQuery.toLowerCase());
    })
  );

  let selectedOption = $derived(options.find((o: { value: any; label: string }) => o.value === value));

  function selectOption(val: any) {
    value = val;
    isOpen = false;
    searchQuery = '';
  }

  function clickOutsideAction(node: HTMLElement) {
    const handleClick = (event: MouseEvent) => {
      if (isOpen && node && !node.contains(event.target as Node)) {
        isOpen = false;
        searchQuery = '';
      }
    };
    document.addEventListener('mousedown', handleClick, true);
    return {
      destroy() {
        document.removeEventListener('mousedown', handleClick, true);
      }
    };
  }

  function toggleOpen() {
    isOpen = !isOpen;
    if (!isOpen) {
      searchQuery = '';
    }
  }

  function autoFocusAction(node: HTMLElement) {
    node.focus();
    return {
      destroy() {}
    };
  }
</script>

<div class="searchable-select" style="width: {width}" use:clickOutsideAction>
  <button 
    class="select-trigger" 
    class:open={isOpen}
    onclick={toggleOpen} 
    type="button"
  >
    <span class="selected-text">{selectedOption ? selectedOption.label : placeholder}</span>
    <ChevronDown size={16} class="chevron {isOpen ? 'rotated' : ''}" />
  </button>
  
  {#if isOpen}
    <div class="dropdown-menu">
      <div class="search-box">
        <span class="icon"><Search size={14} /></span>
        <input 
          type="text" 
          placeholder="Cari kategori..." 
          bind:value={searchQuery}
          onclick={(e) => e.stopPropagation()}
          onkeydown={(e) => { if (e.key === 'Escape') isOpen = false; }}
          onfocus={(e) => e.currentTarget.select()}
          use:autoFocusAction
        />
      </div>
      <div class="options-list">
        {#each filteredOptions as option}
          <button 
            class="option" 
            class:active={value === option.value}
            onclick={() => selectOption(option.value)}
            type="button"
          >
            {#if value === option.value}
              <Check size={14} class="check-icon" />
            {:else}
              <span class="placeholder-icon"></span>
            {/if}
            <span class="option-label">{option.label}</span>
          </button>
        {/each}
        {#if filteredOptions.length === 0}
          <div class="empty-state">Tidak ada hasil cocok.</div>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .searchable-select {
    position: relative;
    display: inline-block;
  }

  .select-trigger {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 10px 14px;
    background: #0f172a;
    border: 1px solid var(--border);
    color: white;
    border-radius: 8px;
    font-family: inherit;
    font-size: 0.95rem;
    cursor: pointer;
    transition: all 0.2s;
  }

  .select-trigger:hover {
    border-color: var(--text-secondary);
  }

  .select-trigger.open {
    border-color: var(--primary);
    box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
  }

  .selected-text {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  :global(.chevron) {
    color: var(--text-secondary);
    transition: transform 0.2s;
    flex-shrink: 0;
  }

  :global(.chevron.rotated) {
    transform: rotate(180deg);
  }

  .dropdown-menu {
    position: absolute;
    top: calc(100% + 8px);
    left: 0;
    width: 100%;
    background: #1e293b;
    border: 1px solid var(--border);
    border-radius: 8px;
    box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5), 0 8px 10px -6px rgba(0, 0, 0, 0.3);
    z-index: 100;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .search-box {
    position: relative;
    padding: 10px;
    border-bottom: 1px solid var(--border);
    background: #0f172a;
  }

  .search-box .icon {
    position: absolute;
    left: 20px;
    top: 50%;
    transform: translateY(-50%);
    color: var(--text-secondary);
    display: flex;
  }

  .search-box input {
    width: 100%;
    padding: 8px 10px 8px 32px;
    background: #1e293b;
    border: 1px solid var(--border);
    border-radius: 6px;
    color: white;
    font-size: 0.9rem;
    outline: none;
    transition: border-color 0.2s;
  }

  .search-box input:focus {
    border-color: var(--primary);
  }

  .options-list {
    max-height: 250px;
    overflow-y: auto;
    padding: 6px;
  }

  .options-list::-webkit-scrollbar {
    width: 6px;
  }
  .options-list::-webkit-scrollbar-track {
    background: transparent;
  }
  .options-list::-webkit-scrollbar-thumb {
    background: rgba(148, 163, 184, 0.2);
    border-radius: 10px;
  }

  .option {
    display: flex;
    align-items: center;
    width: 100%;
    padding: 8px 10px;
    background: transparent;
    border: none;
    color: #e2e8f0;
    font-size: 0.95rem;
    text-align: left;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.1s;
    gap: 8px;
  }

  .option:hover {
    background: rgba(99, 102, 241, 0.1);
  }

  .option.active {
    background: rgba(99, 102, 241, 0.2);
    color: var(--primary);
    font-weight: 600;
  }

  :global(.check-icon) {
    color: var(--primary);
    flex-shrink: 0;
  }

  .placeholder-icon {
    width: 14px;
    height: 14px;
    flex-shrink: 0;
  }

  .option-label {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .empty-state {
    padding: 16px;
    text-align: center;
    color: var(--text-secondary);
    font-size: 0.9rem;
  }
</style>
