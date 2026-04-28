import { writable } from 'svelte/store';

export const wsStatus = writable('disconnected');
export const wsEvents = writable([]);

export class WebSocketService {
  constructor() {
    this.socket = null;
    this.retryCount = 0;
    this.maxRetries = 5;
  }
  
  connect(token) {
    if (typeof window === 'undefined') return;
    
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const url = `${protocol}://${window.location.host}/api/ws?token=${token}`;
    
    this.socket = new WebSocket(url);
    
    this.socket.onopen = () => {
      wsStatus.set('connected');
      this.retryCount = 0;
      console.log('WebSocket connected');
    };
    
    this.socket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        wsEvents.update(events => [data, ...events].slice(0, 50));
        console.log('WS event:', data);
      } catch (e) {
        console.error('Invalid WS message:', e);
      }
    };
    
    this.socket.onclose = () => {
      wsStatus.set('disconnected');
      this.reconnect(token);
    };
    
    this.socket.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  }
  
  reconnect(token) {
    if (this.retryCount < this.maxRetries) {
      this.retryCount++;
      const delay = Math.min(1000 * Math.pow(2, this.retryCount), 30000);
      console.log(`Reconnecting in ${delay}ms... (${this.retryCount})`);
      setTimeout(() => this.connect(token), delay);
    }
  }
  
  disconnect() {
    if (this.socket) {
      this.socket.close();
    }
  }
}
