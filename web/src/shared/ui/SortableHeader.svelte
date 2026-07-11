<script lang="ts">
  let {
    label,
    column,
    sortColumn = '',
    sortDirection = 'asc',
    onsort,
    align = 'left',
    class: className = '',
  }: {
    label: string;
    column: string;
    sortColumn?: string;
    sortDirection?: 'asc' | 'desc' | 'ASC' | 'DESC';
    onsort: (col: string) => void;
    align?: 'left' | 'right';
    class?: string;
  } = $props();

  const isActive = $derived(sortColumn === column);
  const normalizedDirection = $derived(
    sortDirection === 'ASC' ? 'asc' : sortDirection === 'DESC' ? 'desc' : sortDirection
  );
  const ariaSort = $derived(
    !isActive ? 'none' as const : normalizedDirection === 'asc' ? 'ascending' as const : 'descending' as const
  );
</script>

<button
  type="button"
  class="flex items-center gap-1 hover:text-primary transition-colors {align === 'right' ? 'justify-end w-full' : ''}"
  onclick={() => onsort(column)}
  aria-sort={ariaSort}
>
  {label}
  {#if isActive}
    <span>{normalizedDirection === 'asc' ? '▲' : '▼'}</span>
  {/if}
</button>
