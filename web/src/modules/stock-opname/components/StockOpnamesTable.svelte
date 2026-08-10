<script lang="ts">
  import { Button, Badge, Skeleton } from '$shared/ui';
  import { ClipboardList } from 'lucide-svelte';
  import type { StockOpnameSession } from '../types';
  import { STOCK_OPNAME_STATUS_LABELS } from '../types';
  import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
  import { labels } from '$shared/i18n';

  let {
    sessions = [],
    loading = false,
    searchQuery = '',
    canExport = false,
    onview = () => {},
    onexport = () => {},
  }: {
    sessions?: StockOpnameSession[];
    loading?: boolean;
    searchQuery?: string;
    canExport?: boolean;
    onview?: (session: StockOpnameSession) => void;
    onexport?: (session: StockOpnameSession) => void;
  } = $props();

  function getStatusVariant(status: string): 'default' | 'success' | 'warning' | 'danger' | 'primary' | 'muted' {
    switch (status) {
      case 'draft':
        return 'muted';
      case 'open':
      case 'counting':
        return 'primary';
      case 'verification':
      case 'needs_recount':
        return 'warning';
      case 'approved':
      case 'posted':
        return 'success';
      case 'closed':
        return 'muted';
      case 'cancelled':
        return 'danger';
      default:
        return 'default';
    }
  }

  function getScopeLabel(session: StockOpnameSession): string {
    return session.scope_name ? `${session.scope_type} · ${session.scope_name}` : `${session.scope_type} #${session.scope_id}`;
  }
</script>

{#if loading}
  <div class="overflow-x-auto">
    <table class="w-full" style="table-layout: fixed;" aria-busy="true" aria-label={labels.loadingStockOpnameSessions}>
      <colgroup>
        <col style="width: 15%;" />
        <col style="width: 26%;" />
        <col style="width: 14%;" />
        <col style="width: 8%;" />
        <col style="width: 26%;" />
        <col style="width: 11%;" />
      </colgroup>
      <thead><tr><th>{labels.sessionLabel}</th><th>{labels.scopeLabel}</th><th>{labels.statusLabel}</th><th>{labels.blind}</th><th>{labels.createdAtShort}</th><th>{labels.actionsLabel}</th></tr></thead>
      <tbody>{#each Array(5) as _}<tr>{#each Array(6) as _}<td><Skeleton class="h-4 w-20" /></td>{/each}</tr>{/each}</tbody>
    </table>
  </div>
{:else if sessions.length === 0}
  <div class="flex flex-col items-center justify-center py-12 text-text-muted" role="status">
    <ClipboardList class="w-12 h-12 mb-3" aria-hidden="true" />
    <p class="text-text-primary font-medium">{labels.noStockOpnameSessions}</p>
    {#if searchQuery}
      <p class="text-sm text-text-muted mt-1">{labels.tryAdjustingYourSearch}</p>
    {/if}
  </div>
{:else}
  <div class="overflow-x-auto">
    <table class="w-full min-w-[800px]" style="table-layout: fixed;" role="grid" aria-label={labels.stockOpnameSessions}>
      <colgroup>
        <col style="width: 15%;" />
        <col style="width: 26%;" />
        <col style="width: 14%;" />
        <col style="width: 8%;" />
        <col style="width: 26%;" />
        <col style="width: 11%;" />
      </colgroup>
      <thead class="bg-muted/50">
        <tr class="border-b text-left text-sm text-text-muted">
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">{labels.sessionLabel}</th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">{labels.scopeLabel}</th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">{labels.statusLabel}</th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">{labels.blind}</th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">{labels.createdAtShort}</th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap text-right" scope="col">{labels.actionsLabel}</th>
        </tr>
      </thead>
      <tbody>
          {#each sessions as session (session.id)}
            <tr class="border-b border-border transition-colors hover:bg-muted/50 cursor-pointer" onclick={() => onview(session)}>
              <td class="px-4 py-3 text-sm font-medium max-w-0"><span class="truncate block">{session.session_number}</span></td>
              <td class="px-4 py-3 text-sm text-text-secondary max-w-0"><span class="truncate block" title={getScopeLabel(session)}>{getScopeLabel(session)}</span></td>
              <td class="px-4 py-3 whitespace-nowrap">
                <Badge variant={getStatusVariant(session.status)} size="sm">{STOCK_OPNAME_STATUS_LABELS[session.status] || session.status}</Badge>
              </td>
              <td class="px-4 py-3 text-sm text-text-secondary whitespace-nowrap">{session.blind_count ? labels.yes : labels.no}</td>
              <td class="px-4 py-3 text-sm text-text-secondary tabular-nums whitespace-nowrap">{formatDateTimeInJakarta(session.created_at)}</td>
              <td class="px-4 py-3 text-right whitespace-nowrap">
                {#if canExport}
                  <Button variant="ghost" size="sm" onclick={(e) => { e.stopPropagation(); onexport(session); }}>{labels.export}</Button>
                {/if}
              </td>
            </tr>
          {/each}
      </tbody>
    </table>
  </div>
{/if}
