import { writable } from 'svelte/store';
import { refreshTokenSilently } from '$modules/auth';

class WebSocketService {
  status = writable<'disconnected' | 'connecting' | 'connected' | 'error'>('disconnected');
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  private eventHandlers: Record<string, ((data: unknown) => void)[]> = {};
  private disconnectRequested = false;

  connect(token: string) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return;
    }
    if (this.ws) {
      console.log('[WebSocket] Existing ws state:', this.ws.readyState, '| closing before reconnect');
      this.ws.close();
      this.ws = null;
    }

    this.disconnectRequested = false;
    this.reconnectAttempts = 0;
    this.status.set('connecting');

    try {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const backendPort = String(__BACKEND_PORT__) || '9095';
      const backendHost = `${window.location.hostname}:${backendPort}`;
      const wsUrl = `${protocol}//${backendHost}/ws`;

      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        this.ws?.send(JSON.stringify({ type: 'auth', token }));
        this.status.set('connected');
        this.reconnectAttempts = 0;
        this.emit('connection', { status: 'connected' });
      };

      this.ws.onmessage = (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data);
          this.emit(data.type || 'message', data.payload || data);
        } catch (e) {
          console.error('[WebSocket] Parse error:', e);
        }
      };

      this.ws.onclose = async (event: CloseEvent) => {
        console.log('[WebSocket] Closed. Code:', event.code, '| Reason:', event.reason, '| Was clean:', event.wasClean);
        this.status.set('disconnected');
        this.emit('disconnection', { status: 'disconnected', code: event.code });

        if (this.disconnectRequested) {
          console.log('[WebSocket] Close was requested, not reconnecting');
          return;
        }

        if (this.reconnectAttempts < this.maxReconnectAttempts) {
          this.reconnectAttempts++;
          console.log(`[WebSocket] Reconnect attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts}`);

          try {
            await refreshTokenSilently();
            const freshToken = sessionStorage.getItem('access_token');
            if (!freshToken) {
              console.warn('[WebSocket] No token available after refresh, stopping reconnects');
              this.status.set('error');
              return;
            }
            const delay = Math.min(2000 * this.reconnectAttempts, 30000);
            console.log(`[WebSocket] Reconnecting in ${delay}ms`);
            this.reconnectTimeout = setTimeout(() => this.connect(freshToken), delay);
          } catch (e) {
            console.error('[WebSocket] Token refresh failed during reconnect:', e);
            this.status.set('error');
          }
        } else {
          console.warn('[WebSocket] Max reconnect attempts reached, giving up');
          this.status.set('error');
        }
      };

      this.ws.onerror = (err: Event) => {
        console.error('[WebSocket] Error event:', err);
        this.status.set('error');
        if (!this.disconnectRequested && this.ws) {
          if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
            this.ws.close();
          }
        }
      };
    } catch (e) {
      this.status.set('error');
      console.error('[WebSocket] Connection failed:', e);
    }
  }

  disconnect() {
    this.disconnectRequested = true;
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.status.set('disconnected');
  }

  emit(event: string, data: unknown) {
    if (this.eventHandlers[event]) {
      this.eventHandlers[event].forEach(callback => callback(data));
    }
  }

  on(event: string, callback: (data: unknown) => void) {
    if (!this.eventHandlers[event]) {
      this.eventHandlers[event] = [];
    }
    this.eventHandlers[event].push(callback);
    
    return () => {
      this.eventHandlers[event] = this.eventHandlers[event].filter(cb => cb !== callback);
    };
  }

  send(type: string, payload: Record<string, unknown>) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type, ...payload }));
    }
  }
}

let instance: WebSocketService | null = null;

export function useWebSocket() {
  if (!instance) {
    instance = new WebSocketService();
  }
  return instance;
}

export default WebSocketService;
