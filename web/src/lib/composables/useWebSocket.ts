import { writable } from 'svelte/store';
import { dev } from '$app/environment';

class WebSocketService {
  status = writable('disconnected');
  private ws = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectTimeout = null;
  private eventHandlers = {};

  connect(token) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return;
    }

    this.status.set('connecting');
    
    try {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const frontendPort = import.meta.env.VITE_FRONTEND_PORT || '5173';
      const backendPort = import.meta.env.VITE_BACKEND_PORT || '9095';
      const backendHost = window.location.host === `localhost:${frontendPort}` ? `localhost:${backendPort}` : window.location.host;
      const wsUrl = `${protocol}//${backendHost}/ws?token=${encodeURIComponent(token || '')}`;
      
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        this.status.set('connected');
        this.reconnectAttempts = 0;
        this.emit('connection', { status: 'connected' });
      };

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          this.emit(data.type || 'message', data);
        } catch (e) {
          console.error('WS parse error:', e);
        }
      };

      this.ws.onclose = () => {
        this.status.set('disconnected');
        this.emit('disconnection', { status: 'disconnected' });
        
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
          this.reconnectAttempts++;
          this.reconnectTimeout = setTimeout(() => this.connect(token), 2000 * this.reconnectAttempts);
        }
      };

      this.ws.onerror = (err) => {
        console.error('WebSocket error:', err);
        this.status.set('error');
      };
    } catch (e) {
      this.status.set('error');
      console.error('WebSocket connection failed:', e);
    }
  }

  disconnect() {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.status.set('disconnected');
  }

  emit(event, data) {
    if (this.eventHandlers[event]) {
      this.eventHandlers[event].forEach(callback => callback(data));
    }
  }

  on(event, callback) {
    if (!this.eventHandlers[event]) {
      this.eventHandlers[event] = [];
    }
    this.eventHandlers[event].push(callback);
    
    return () => {
      this.eventHandlers[event] = this.eventHandlers[event].filter(cb => cb !== callback);
    };
  }

  send(type, payload) {
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
