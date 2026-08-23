import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'TransactionsPage.svelte'), 'utf-8');
}

describe('TransactionsPage.svelte source-structure guards', () => {
  const src = getSource();

  it('imports extracted child components', () => {
    expect(src).toContain("import TransactionFilters from './TransactionFilters.svelte'");
    expect(src).toContain("import TransactionTable from './TransactionTable.svelte'");
    expect(src).toContain("import TransactionDrawer from './TransactionDrawer.svelte'");
  });

  it('renders child components in template', () => {
    expect(src).toContain('<TransactionFilters');
    expect(src).toContain('<TransactionTable');
    expect(src).toContain('<TransactionDrawer');
  });

  it('imports Jakarta time utilities', () => {
    expect(src).toContain("import { getTodayInJakarta, getDateNDaysAgoInJakarta, JAKARTA_OFFSET_MS } from '$shared/utils/jakartaTime'");
  });

  it('imports useSalesStore from store', () => {
    expect(src).toContain("import { useSalesStore } from '../stores/sales-store.svelte'");
  });

  it('imports FindTransaction and Permissions for the lookup tab', () => {
    expect(src).toContain("import FindTransaction from './FindTransaction.svelte'");
    expect(src).toContain("import { Permissions } from '$shared/constants/permissions'");
  });

  it('defaults the active tab to My Transactions', () => {
    expect(src).toContain("let activeTab = $state<'mine' | 'lookup'>('mine')");
  });

  it('only offers the Find Transaction tab to sale.lookup holders', () => {
    expect(src).toContain('const canLookup = $derived(rbac.can(Permissions.sale.lookup))');
    expect(src).toContain('{#if canLookup}');
  });

  it('hides the tab bar entirely for report.view holders (single all-cashier view)', () => {
    expect(src).toContain('const canAccessAll = $derived(rbac.can(Permissions.report.view))');
    expect(src).toContain('{#if !canAccessAll}');
  });

  it('renders the Find Transaction panel when the lookup tab is active', () => {
    expect(src).toContain('activeTab === \'lookup\'');
    expect(src).toContain('<FindTransaction />');
  });

  it('imports auth store and RBAC', () => {
    expect(src).toContain("import { useAuthStore } from '$modules/auth'");
    expect(src).toContain("import { useRBAC } from '$shared/composables/useRBAC.svelte'");
  });

  it('sets cashierId filter for cashier role', () => {
    expect(src).toContain('store.cashierId');
  });

  it('imports createQueryManager', () => {
    expect(src).toContain("import { createQueryManager } from '../lib/query-manager'");
  });

  it('initializes store with default dates', () => {
    expect(src).toContain('store.startDate = ');
    expect(src).toContain('store.endDate = ');
    expect(src).toContain("store.dateRange = 'last30d'");
  });

  it('has toggleSort and handlePageChange', () => {
    expect(src).toContain('function toggleSort');
    expect(src).toContain('function handlePageChange');
  });

  it('has openTransactionDetails and closeTransactionDrawer', () => {
    expect(src).toContain('function openTransactionDetails');
    expect(src).toContain('function closeTransactionDrawer');
  });

  it('has handleKeydown', () => {
    expect(src).toContain('function handleKeydown');
  });

  it('has onMount lifecycle', () => {
    expect(src).toContain('onMount(');
  });

  it('has $effect for filter watching', () => {
    expect(src).toContain('$effect(() => {');
  });

  it('creates query manager', () => {
    expect(src).toContain('const qm = createQueryManager');
  });

  it('calls store.loadPaymentMethods in onMount', () => {
    expect(src).toContain('store.loadPaymentMethods()');
  });

  it('binds filter props to store values', () => {
    expect(src).toContain("bind:searchQuery={store.searchQuery}");
    expect(src).toContain("bind:startDate={store.startDate}");
    expect(src).toContain('bind:showDatePicker');
  });

  it('imports shift store, router, toast and labels for shift guard', () => {
    expect(src).toContain("import { useShiftStore } from '$modules/shifts'");
    expect(src).toContain("import { goto, subscribe as subscribeRoute } from '$app/router'");
    expect(src).toContain("import { toast } from '$shared/stores/toast.svelte'");
    expect(src).toContain("import { labels } from '$shared/i18n'");
  });

  it('loads active shift from shiftStore on mount', () => {
    expect(src).toContain('shiftStore.loadActiveShift()');
  });

  it('redirects cashier without an active shift to /shifts', () => {
    expect(src).toContain('shiftStore.activeShift');
    expect(src).toContain("goto('/shifts')");
    expect(src).toContain('toastMustOpenShiftFirst');
  });

  it('imports RefreshCw, useWebSocket, Button for refresh + banner', () => {
    expect(src).toContain("import { RefreshCw } from 'lucide-svelte'");
    expect(src).toContain("import { useWebSocket } from '$shared/api/websocket'");
    expect(src).toContain("import { Button } from '$shared/ui'");
  });

  it('has refresh + viewNew and resets to page 0', () => {
    expect(src).toContain('function refresh');
    expect(src).toContain('function viewNew');
    expect(src).toContain('store.page = 0');
  });

  it('tracks last-updated timestamp and refreshing state', () => {
    expect(src).toContain('let lastUpdated');
    expect(src).toContain('let refreshing');
    expect(src).toContain('let newTxnCount');
    expect(src).toContain('let newTxnSince');
    expect(src).toContain('function jakartaHHMM');
  });

  it('subscribes to sale_created only for the all-sales (manager) view', () => {
    expect(src).toContain("ws.on('sale_created'");
    expect(src).toContain('if (!canAccessAll) return;');
  });

  it('renders the refresh button and Updated label', () => {
    expect(src).toContain('labels.refresh');
    expect(src).toContain('labels.updated');
    expect(src).toContain('labels.transaction');
    expect(src).toContain('onclick={refresh}');
  });

  it('renders the manager new-transactions banner', () => {
    expect(src).toContain('canAccessAll && newTxnCount > 0');
    expect(src).toContain('labels.newTransactionsSince');
    expect(src).toContain('onclick={viewNew}');
  });
});
