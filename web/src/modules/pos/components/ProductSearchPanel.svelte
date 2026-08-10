<script lang="ts">
  import { SearchBar } from '$shared/ui';
  import { labels } from '$shared/i18n';

  let {
    searchQuery = $bindable(''),
    onsearchsubmit,
  }: {
    searchQuery: string;
    onsearchsubmit?: () => void;
  } = $props();
</script>

<div class="card p-4">
  <div class="flex flex-wrap items-center gap-2">
    <kbd class="px-1.5 py-0.5 text-[10px] font-medium text-primary/60 bg-primary-subtle/30 rounded border border-primary/20 select-none">F2</kbd>
    <span class="flex items-center gap-1 text-text-muted text-[11px]">
      <kbd class="px-1.5 py-0.5 text-[10px] font-medium text-primary/60 bg-primary-subtle/30 rounded border border-primary/20 select-none">↑↓</kbd>
      <span>{labels.posSelectProductHint}</span>
      <kbd class="px-1.5 py-0.5 text-[10px] font-medium text-primary/60 bg-primary-subtle/30 rounded border border-primary/20 select-none">Enter</kbd>
      <span>{labels.posAddToCartHint}</span>
    </span>
    <div class="flex-1">
      <SearchBar bind:value={searchQuery} placeholder={labels.searchByNameSkuBarcode} id="pos-search-input" onsubmit={onsearchsubmit} />
    </div>
  </div>
</div>

<style></style>
