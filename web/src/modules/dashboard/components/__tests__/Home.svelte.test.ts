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
    expect(src).toContain("Today's Revenue");
    expect(src).toContain("Transactions");
    expect(src).toContain("Total Products");
    expect(src).toContain("Low Stock Alerts");
  });

  it('has Quick Access modules section', () => {
    expect(src).toContain("Quick Access");
    expect(src).toContain("Point of Sale");
    expect(src).toContain("Inventory");
    expect(src).toContain("Reports");
    expect(src).toContain("Administration");
  });

  it('has WebSocket live status indicator', () => {
    expect(src).toContain('wsConnected ? \'Live\' : \'Offline\'');
  });
});
