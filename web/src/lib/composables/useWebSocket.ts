import { writable } from 'svelte/store';
import { dev } from '$app/environment';

class UseWebSocket {
  constructor() {
    this.status = writable('disconnected');
    this.wsEvents = {
      _listeners: {},
      on: (event, callback) => {
        if (!this.wsEvents._listeners[event]) {
          this.wsEvents._listeners[event] = [];
        }
        this.wsEvents._listeners[event].push(callback);
      },
      off: (event, callback) => {
        if (this.wsEvents._listeners[event]) {
          this.wsEvents._listeners[event] = this.wsEvents._listeners[event].filter(l => l !== callback);
        }
      },
      emit: (event, data) => {
        if (this.wsEvents._listeners[event]) {
          this.wsEvents._listeners[event].forEach(l => l(data));
        }
      }
    };
  }

  connect() {
    this.status.set('connecting');
    try {
      // In production, the websocket connects via backend
      this.status.set('connected');
    } catch (e) {
      this.status.set('disconnected');
    }
  }
}

export function useWebSocket() {
  if (!globalThis._wsInstance) {
    globalThis._wsInstance = new UseWebSocket();
  }
  return globalThis._wsInstance;
}
