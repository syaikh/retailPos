<script lang="ts">
  import { Badge } from '$shared/ui';
  import type { PreviewRow } from '$shared/types/import-export';

  let {
    rows = [] as PreviewRow[],
    columns = [] as string[],
    maxHeight = 'h-72',
  }: {
    rows?: PreviewRow[];
    columns?: string[];
    maxHeight?: string;
  } = $props();

  function statusBadge(row: PreviewRow) {
    switch (row.status) {
      case 'insert':
        return { label: 'Insert', class: 'bg-emerald-500/10 text-emerald-400' };
      case 'update':
        return { label: 'Update', class: 'bg-amber-500/10 text-amber-400' };
      case 'error':
        return { label: 'Error', class: 'bg-rose-500/10 text-rose-400' };
    }
  }

  function cellValue(row: PreviewRow, col: string): string {
    const newVal = (row as any).new_values ?? row.newValues;
    const oldVal = (row as any).old_values ?? row.oldValues;
    if (row.status === 'update') {
      const ov = oldVal?.[col];
      const nv = newVal[col];
      if (ov !== undefined && ov !== nv) {
        return `${ov} → ${nv}`;
      }
    }
    return newVal[col] ?? '';
  }
</script>

{#if rows.length > 0}
  <div class="overflow-x-auto border border-border rounded-lg {maxHeight} overflow-y-auto">
    <table class="w-full text-xs">
      <thead class="bg-surface-subtle sticky top-0 z-10 border-b border-border">
        <tr>
          <th class="px-3 py-2 text-left font-semibold text-text-muted uppercase tracking-wider w-16">Status</th>
          <th class="px-3 py-2 text-left font-semibold text-text-muted uppercase tracking-wider w-12">Row</th>
          {#each columns as col}
            <th class="px-3 py-2 text-left font-semibold text-text-muted uppercase tracking-wider whitespace-nowrap">{col}</th>
          {/each}
        </tr>
      </thead>
      <tbody class="divide-y divide-border/60">
        {#each rows as row}
          <tr class="hover:bg-surface-hover/30 transition-colors">
            <td class="px-3 py-1.5">
              {#if row.status === 'error'}
                <span class="inline-flex items-center gap-1 text-rose-400 font-medium">
                  <span class="w-1.5 h-1.5 rounded-full bg-rose-400 shrink-0"></span>
                  Error
                </span>
              {:else}
                <span class="inline-flex items-center gap-1 {row.status === 'insert' ? 'text-emerald-400' : 'text-amber-400'} font-medium">
                  <span class="w-1.5 h-1.5 rounded-full {row.status === 'insert' ? 'bg-emerald-400' : 'bg-amber-400'} shrink-0"></span>
                  {row.status === 'insert' ? 'Insert' : 'Update'}
                </span>
              {/if}
            </td>
            <td class="px-3 py-1.5 text-text-muted">{(row as any).row_number ?? row.rowNumber}</td>
            {#each columns as col}
              <td class="px-3 py-1.5 text-text-primary truncate max-w-48">
                {cellValue(row, col)}
              </td>
            {/each}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}
