<script lang="ts">
  import { Globe, Check } from 'lucide-svelte';
  import Dropdown from './Dropdown.svelte';
  import { currentLocale, labels, setLocale } from '$shared/i18n';
  import type { Locale } from '$shared/i18n';

  const options: { code: Locale; label: string }[] = [
    { code: 'id', label: 'Bahasa Indonesia' },
    { code: 'en', label: 'English' },
  ];

  let open = $state(false);
</script>

<Dropdown placement="bottom-end" bind:open menuClass="min-w-[160px]">
  {#snippet trigger({ toggle })}
    <button
      type="button"
      onclick={toggle}
      aria-label={labels.switchLanguage}
      title={labels.switchLanguage}
      aria-haspopup="menu"
      aria-expanded={open}
      class="flex items-center gap-1.5 p-2 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
    >
      <Globe size={18} />
      <span class="text-xs font-bold uppercase">{currentLocale()}</span>
    </button>
  {/snippet}

  {#snippet content({ close })}
    <div role="menu" aria-label={labels.switchLanguage}>
      {#each options as opt}
        <button
          type="button"
          role="menuitem"
          class="w-full flex items-center gap-3 px-3 py-2 text-sm transition-colors text-text-secondary hover:bg-surface-hover hover:text-text-primary"
          onclick={() => { setLocale(opt.code); close(); }}
        >
          <span class="flex-1 text-left">{opt.label}</span>
          {#if currentLocale() === opt.code}
            <Check size={14} class="text-primary-light" />
          {/if}
        </button>
      {/each}
    </div>
  {/snippet}
</Dropdown>
