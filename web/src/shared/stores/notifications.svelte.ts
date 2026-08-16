import { writable, derived } from 'svelte/store';
import { JAKARTA_OFFSET_MS } from '$shared/utils/jakartaTime';
import { labels, t, formatLocaleDate } from '$shared/i18n';

export type NotificationType = 'low_stock' | 'sale_created' | 'stock_update' | 'product_updated' | 'po_received' | 'so_created' | 'so_submitted' | 'so_approved' | 'so_rejected' | 'so_needs_recount' | 'so_cancelled';

export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  description: string;
  timestamp: Date;
  read: boolean;
  navigateTo?: string;
}

const MAX_NOTIFICATIONS = 50;

function createNotificationStore() {
  const { subscribe, update } = writable<Notification[]>([]);

  const unreadCount = derived({ subscribe }, ($notifications) =>
    $notifications.filter(n => !n.read).length
  );

  function push(n: Omit<Notification, 'id' | 'timestamp' | 'read'>) {
    update(notifs => {
      const updated = [{
        ...n,
        id: crypto.randomUUID(),
        timestamp: new Date(),
        read: false,
      }, ...notifs];
      return updated.slice(0, MAX_NOTIFICATIONS);
    });
  }

  function markAsRead(id: string) {
    update(notifs => notifs.map(n => n.id === id ? { ...n, read: true } : n));
  }

  function markAllRead() {
    update(notifs => notifs.map(n => ({ ...n, read: true })));
  }

  function clear() {
    update(() => []);
  }

  return { subscribe, unreadCount, push, markAsRead, markAllRead, clear };
}

export const notifications = createNotificationStore();

export function formatRelativeTime(date: Date): string {
  const now = Date.now();
  const diffMs = now - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return labels.justNow;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return t('minutesAgoShort', { n: diffMin });
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return t('hoursAgoShort', { n: diffHour });
  const diffDay = Math.floor(diffHour / 24);
  if (diffDay < 7) return t('daysAgoShort', { n: diffDay });
  const jakartaDate = new Date(date.getTime() + JAKARTA_OFFSET_MS);
  return `${jakartaDate.getUTCDate()} ${formatLocaleDate(jakartaDate, { month: 'short', timeZone: 'UTC' })}`;
}

export function getNotificationIcon(type: NotificationType): string {
  switch (type) {
    case 'low_stock': return '🔴';
    case 'sale_created': return '🛒';
    case 'stock_update': return '📦';
    case 'product_updated': return '✏️';
    case 'po_received': return '📥';
    case 'so_created': return '📋';
    case 'so_submitted': return '✅';
    case 'so_approved': return '👍';
    case 'so_rejected': return '❌';
    case 'so_needs_recount': return '🔄';
    case 'so_cancelled': return '🚫';
  }
}

export const STOCK_OPNAME_VIEW_PERMISSION = 'stock_opname.view';

export function canReceiveStockOpnameNotifications(permissions: string[] | undefined | null): boolean {
  return !!permissions?.includes(STOCK_OPNAME_VIEW_PERMISSION);
}
