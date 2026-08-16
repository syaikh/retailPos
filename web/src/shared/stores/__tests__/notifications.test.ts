import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { Notification } from '../notifications.svelte.ts';
import type { notifications as NotificationsStore } from '../notifications.svelte.ts';

describe('notifications store', () => {
  let notifications: typeof NotificationsStore;

  beforeEach(() => {
    vi.resetModules();
  });

  it('returns expected API shape', async () => {
    const { notifications: n } = await import('../notifications.svelte');
    notifications = n;
    expect(notifications).toHaveProperty('subscribe');
    expect(notifications).toHaveProperty('unreadCount');
    expect(notifications).toHaveProperty('push');
    expect(notifications).toHaveProperty('markAsRead');
    expect(notifications).toHaveProperty('markAllRead');
    expect(notifications).toHaveProperty('clear');
  });

  it('push adds notification with id and timestamp', async () => {
    const { notifications: n } = await import('../notifications.svelte');
    notifications = n;
    notifications.push({ type: 'low_stock', title: 'Low Stock', description: 'Item low' });
    let toasts: Notification[] = [];
    notifications.subscribe((value) => { toasts = value; })();
    expect(toasts).toHaveLength(1);
    expect(toasts[0].id).toBeDefined();
    expect(toasts[0].timestamp).toBeInstanceOf(Date);
    expect(toasts[0].read).toBe(false);
  });

  it('unreadCount is derived correctly', async () => {
    const { notifications: n } = await import('../notifications.svelte');
    notifications = n;
    notifications.push({ type: 'low_stock', title: 'Low1', description: '' });
    notifications.push({ type: 'sale_created', title: 'Sale1', description: '' });
    let count = 0;
    notifications.unreadCount.subscribe((v) => { count = v; })();
    expect(count).toBe(2);
  });

  it('markAsRead sets read to true', async () => {
    const { notifications: n } = await import('../notifications.svelte');
    notifications = n;
    notifications.push({ type: 'low_stock', title: 'Low', description: '' });
    let toasts: Notification[] = [];
    notifications.subscribe((value) => { toasts = value; })();
    notifications.markAsRead(toasts[0].id);
    notifications.subscribe((value) => { toasts = value; })();
    expect(toasts[0].read).toBe(true);
  });

  it('markAllRead sets all to read', async () => {
    const { notifications: n } = await import('../notifications.svelte');
    notifications = n;
    notifications.push({ type: 'low_stock', title: 'Low', description: '' });
    notifications.push({ type: 'sale_created', title: 'Sale', description: '' });
    notifications.markAllRead();
    let toasts: Notification[] = [];
    notifications.subscribe((value) => { toasts = value; })();
    expect(toasts.every(n => n.read)).toBe(true);
  });

  it('clear removes all notifications', async () => {
    const { notifications: n } = await import('../notifications.svelte');
    notifications = n;
    notifications.push({ type: 'low_stock', title: 'Low', description: '' });
    notifications.clear();
    let toasts: Notification[] = [];
    notifications.subscribe((value) => { toasts = value; })();
    expect(toasts).toHaveLength(0);
  });

  it('formatRelativeTime returns just now for seconds', async () => {
    const { formatRelativeTime } = await import('../notifications.svelte');
    const result = formatRelativeTime(new Date(Date.now() - 30000));
    expect(result).toBe('Baru saja');
  });

  it('formatRelativeTime returns minutes for under an hour', async () => {
    const { formatRelativeTime } = await import('../notifications.svelte');
    const result = formatRelativeTime(new Date(Date.now() - 5 * 60 * 1000));
    expect(result).toBe('5m lalu');
  });

  it('getNotificationIcon returns correct icons', async () => {
    const { getNotificationIcon } = await import('../notifications.svelte');
    expect(getNotificationIcon('low_stock')).toBe('🔴');
    expect(getNotificationIcon('sale_created')).toBe('🛒');
    expect(getNotificationIcon('stock_update')).toBe('📦');
    expect(getNotificationIcon('product_updated')).toBe('✏️');
    expect(getNotificationIcon('po_received')).toBe('📥');
    expect(getNotificationIcon('so_created')).toBe('📋');
    expect(getNotificationIcon('so_submitted')).toBe('✅');
    expect(getNotificationIcon('so_approved')).toBe('👍');
    expect(getNotificationIcon('so_rejected')).toBe('❌');
    expect(getNotificationIcon('so_needs_recount')).toBe('🔄');
    expect(getNotificationIcon('so_cancelled')).toBe('🚫');
  });

  it('canReceiveStockOpnameNotifications allows stock_opname.view', async () => {
    const { canReceiveStockOpnameNotifications } = await import('../notifications.svelte');
    expect(canReceiveStockOpnameNotifications(['dashboard.view', 'stock_opname.view'])).toBe(true);
    expect(canReceiveStockOpnameNotifications(['stock_opname.count'])).toBe(false);
    expect(canReceiveStockOpnameNotifications([])).toBe(false);
    expect(canReceiveStockOpnameNotifications(undefined)).toBe(false);
    expect(canReceiveStockOpnameNotifications(null)).toBe(false);
  });
});