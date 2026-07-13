<script lang="ts">
  import { SearchBar } from '$shared/ui';

  let {
    searchQuery = $bindable(''),
    onsearchsubmit,
  }: {
    searchQuery: string;
    onsearchsubmit?: () => void;
  } = $props();
</script>

<div class="card p-4">
  <div class="flex items-center gap-2">
    <kbd class="px-1.5 py-0.5 text-[10px] font-medium text-primary/60 bg-primary-subtle/30 rounded border border-primary/20 select-none">F2</kbd>
    <div class="flex-1">
      <SearchBar bind:value={searchQuery} placeholder="Search by name, SKU, or barcode..." id="pos-search-input" onsubmit={onsearchsubmit} />
    </div>
  </div>
</div>

<style></style>
