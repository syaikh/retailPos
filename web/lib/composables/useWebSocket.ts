import { onMount, onDestroy } from 'svelte';
import { writable, get } from 'svelte/store';
import { auth } from '$lib/stores/auth';
import { ui } from '$lib/stores/ui';

export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting';

export interface WSMessage {
  type: string;
  payload: any;
  timestamp: string;
  store_id?: number;
}

// Event bus for global WebSocket events
class EventBus {
  private handlers: Map<string, Set<(data: any) => void>> = new Map();

  on(event: string, handler: (data: any) => void) {
    if (!this.handlers.has(event)) {
      this.handlers.set(event, new Set());
    }
    this.handlers.get(event)!.add(handler);
  }

  off(event: string, handler: (data: any) => void) {
    this.handlers.get(event)?.delete(handler);
  }

  emit(event: string, data: any) {
    this.handlers.get(event)?.forEach(h => {
      try { h(data); } catch(e) { console.error('Event handler error:', e); }
    });
  }
}

export const wsEvents = new EventBus();

export function useWebSocket() {
  const status = writable<ConnectionState>('disconnected');
  const lastError = writable<string | null>(null);
  const reconnectAttempts = writable(0);

  let ws: WebSocket | null = null;
  let pingInterval: number | null = null;
  let reconnectTimer: number | null = null;
  let shouldReconnect = true;

  const connect = () => {
    const authState = get(auth);
    if (!authState.user || !authState.isAuthenticated) {
      console.log('Not authenticated, skipping WebSocket connection');
      return;
    }

    const token = authState.user?.permissions?.length ? btoa(JSON.stringify({ token: 'jwt-from-cookie' })) : '';

    // Use wss in production, ws in development
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = `${protocol}//${window.location.host}/api/ws?token=${authState.user.username}`; // Simplified: backend reads JWT from cookie

    status.set('connecting');
    ws = new WebSocket(url);

    ws.onopen = () => {
      status.set('connected');
      reconnectAttempts.set(0);
      lastError.set(null);
      console.log('WebSocket connected');

      // Start ping
      pingInterval = window.setInterval(() => {
        if (ws?.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'ping' }));
        }
      }, 30000);

      wsEvents.emit('connected', null);
    };

    ws.onmessage = (event) => {
      try {
        const message: WSMessage = JSON.parse(event.data);
        
        // Broadcast global event
        wsEvents.emit(message.type, message.payload);
        wsEvents.emit('message', message);

        // Handle specific event types
        switch (message.type) {
          case 'stock_update':
            wsEvents.emit('stockChanged', message.payload);
            if (message.payload.low_stock) {
              ui.warning(`Stok rendah: ${message.payload.sku} (sisa: ${message.payload.stock})`);
            }
            break;

          case 'sale_created':
            wsEvents.emit('saleCreated', message.payload);
            ui.success(`Penjualan baru: ${message.payload.invoice} (Rp${message.payload.total.toLocaleString()})`);
            break;

          case 'product_updated':
            wsEvents.emit('productUpdated', message.payload);
            break;

          case 'low_stock_alert':
            ui.error(`⚠️ Stok sangat rendah: ${message.payload.name} (sisa: ${message.payload.stock})`);
            wsEvents.emit('lowStock', message.payload);
            break;

          case 'user_online_count':
            ui.store.onlineUsers = message.payload.count;
            break;
        }
      } catch (e) {
        console.error('Error parsing WebSocket message:', e);
      }
    };

    ws.onclose = (event) => {
      status.set('disconnected');
      console.log('WebSocket disconnected:', event.code, event.reason);

      if (pingInterval) {
        clearInterval(pingInterval);
        pingInterval = null;
      }

      wsEvents.emit('disconnected', event);

      if (shouldReconnect) {
        const attempts = get(reconnectAttempts);
        const delay = Math.min(1000 * Math.pow(2, attempts), 30000);

        reconnectAttempts.set(attempts + 1);
        status.set('reconnecting');

        reconnectTimer = window.setTimeout(() => {
          console.log(`Reconnecting... (attempt ${attempts + 1})`);
          connect();
        }, delay);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      lastError.set('Koneksi WebSocket error');
      status.set('disconnected');
    };
  };

  const disconnect = () => {
    shouldReconnect = false;

    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }

    if (pingInterval) {
      clearInterval(pingInterval);
      pingInterval = null;
    }

    if (ws) {
      ws.close();
      ws = null;
    }

    status.set('disconnected');
  };

  const send = (type: string, payload: any) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type, payload }));
    }
  };

  onDestroy(() => {
    disconnect();
  });

  // Auto-connect if authenticated
  const unsubscribe = auth.subscribe((state) => {
    if (state.isAuthenticated && state.user && !ws) {
      setTimeout(() => connect(), 500);
    } else if (!state.isAuthenticated) {
      disconnect();
    }
  });

  onDestroy(unsubscribe);

  return {
    status,
    lastError,
    reconnectAttempts,
    connect,
    disconnect,
    send,
    isConnected: () => ws?.readyState === WebSocket.OPEN
  };
}
