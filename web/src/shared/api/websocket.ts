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

    this.disconnectRequested = false;
    this.reconnectAttempts = 0;
    this.status.set('connecting');
    
    try {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const frontendPort = String(__FRONTEND_PORT__) || '5173';
      const backendPort = String(__BACKEND_PORT__) || '9095';
      const backendHost = window.location.host === `localhost:${frontendPort}` ? `localhost:${backendPort}` : window.location.host;
      const wsUrl = `${protocol}//${backendHost}/ws?token=${encodeURIComponent(token || '')}`;
      
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        this.status.set('connected');
        this.reconnectAttempts = 0;
        this.emit('connection', { status: 'connected' });
      };

      this.ws.onmessage = (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data);
          this.emit(data.type || 'message', data.payload || data);
        } catch (e) {
          console.error('WS parse error:', e);
        }
      };

      this.ws.onclose = async () => {
        this.status.set('disconnected');
        this.emit('disconnection', { status: 'disconnected' });

        if (this.disconnectRequested) return;

        if (this.reconnectAttempts < this.maxReconnectAttempts) {
          this.reconnectAttempts++;
          await refreshTokenSilently();
          const freshToken = sessionStorage.getItem('access_token') || token;
          this.reconnectTimeout = setTimeout(() => this.connect(freshToken), 2000 * this.reconnectAttempts);
        }
      };

      this.ws.onerror = (err: Event) => {
        console.error('WebSocket error:', err);
        this.status.set('error');
      };
    } catch (e) {
      this.status.set('error');
      console.error('WebSocket connection failed:', e);
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

let instance = null;

export function useWebSocket() {
  if (!instance) {
    instance = new WebSocketService();
  }
  return instance;
}

export default WebSocketService;
