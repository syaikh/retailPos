import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'Home.svelte'), 'utf-8');
}

describe('Home.svelte source-structure guards', () => {
  const src = getSource();

  it('imports onMount from svelte', () => {
    expect(src).toContain("import { onMount } from 'svelte'");
  });

  it('imports goto from $app/router', () => {
    expect(src).toContain("import { goto } from '$app/router'");
  });

  it('imports apiFetch from shared/api', () => {
    expect(src).toContain("import { apiFetch } from '$shared/api/http-client'");
  });

  it('imports StatCard from shared/ui', () => {
    expect(src).toContain("import { StatCard } from '$shared/ui'");
  });

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels } from '$shared/i18n'");
  });

  it('uses $state for dashboard data', () => {
    expect(src).toContain('let todaysRevenue = $state');
    expect(src).toContain('let todaysSales = $state');
    expect(src).toContain('let totalProducts = $state');
    expect(src).toContain('let lowStockCount = $state');
    expect(src).toContain('let loading = $state');
    expect(src).toContain('let wsConnected = $state');
  });

  it('uses $derived for revenue subtitle', () => {
    expect(src).toContain('const revSubText = $derived');
  });

  it('has fetchLiveStats function calling dashboard API', () => {
    expect(src).toContain('async function fetchLiveStats');
    expect(src).toContain('/api/dashboard/live');
  });

  it('sets up WebSocket and interval in onMount', () => {
    expect(src).toContain('fetchLiveStats()');
    expect(src).toContain('ws.on');
  });

  it('renders StatCard for revenue, transactions, products, low stock', () => {
    expect(src).toContain('labels.todayRevenue');
    expect(src).toContain('labels.transactionsCard');
    expect(src).toContain('labels.totalProducts');
    expect(src).toContain('labels.lowStockAlerts');
  });

  it('has Quick Access modules section', () => {
    expect(src).toContain('labels.quickAccess');
    expect(src).toContain('labels.pointOfSale');
    expect(src).toContain('labels.inventory');
    expect(src).toContain('labels.reports');
    expect(src).toContain('labels.administration');
  });

  it('has WebSocket live status indicator', () => {
    expect(src).toContain('wsConnected ? labels.live : labels.offline');
  });
});
