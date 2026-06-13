import { describe, it, expect, beforeAll } from 'vitest';
import { readFileSync } from 'fs';
import { fileURLToPath } from 'url';
import { useWebSocket } from '$lib/composables/useWebSocket';

const __filename = fileURLToPath(import.meta.url);

describe('Home.svelte live dashboard', () => {
  let source: string;

  beforeAll(() => {
    source = readFileSync(__filename.replace('Home.svelte.test.ts', 'Home.svelte'), 'utf-8');
  });

  it('has wsConnected state initialized to false', () => {
    expect(source).toContain('let wsConnected = $state(false)');
  });

  it('subscribes to ws status and updates wsConnected', () => {
    expect(source).toContain('ws.status.subscribe');
    expect(source).toContain('wsConnected = status === \'connected\'');
  });

  it('listens to sale_created via ws.on and increments counters', () => {
    expect(source).toContain("ws.on('sale_created'");
    expect(source).toContain('todaysRevenue += data.total');
    expect(source).toContain('todaysSales += 1');
  });

  it('fetches live stats from /api/dashboard/live on mount', () => {
    expect(source).toContain("apiFetch('/api/dashboard/live')");
    expect(source).toContain('todaysRevenue = data.data.todays_revenue');
    expect(source).toContain('todaysSales = data.data.todays_sales');
    expect(source).toContain('totalProducts = data.data.total_products');
    expect(source).toContain('lowStockCount = data.data.low_stock_count');
  });

  it('sets up polling interval for fetchLiveStats', () => {
    expect(source).toContain('setInterval(fetchLiveStats, 30000)');
  });

  it('cleans up ws subscriptions and interval on unmount', () => {
    expect(source).toContain('handlers.forEach((fn) => fn())');
    expect(source).toContain('clearInterval(intervalId)');
  });

  it('renders Live Dashboard header and connection indicator', () => {
    expect(source).toContain("Live Dashboard");
    expect(source).toContain('animate-ping');
    expect(source).toContain("{wsConnected ? 'Live' : 'Offline'}");
  });

  it('renders all four stat cards', () => {
    expect(source).toContain("Today's Revenue");
    expect(source).toContain('Transactions');
    expect(source).toContain('Total Products');
    expect(source).toContain('Low Stock Alerts');
  });

  it('renders Quick Access modules', () => {
    expect(source).toContain("Point of Sale");
    expect(source).toContain('Inventory');
    expect(source).toContain('Reports');
    expect(source).toContain('Administration');
  });

  it('Inventory quick link points to /inventory/products', () => {
    expect(source).toContain("href: '/inventory/products'");
  });
});
