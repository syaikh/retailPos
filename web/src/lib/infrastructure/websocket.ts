export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting';

interface EventHandler {
    (data: any): void;
}

interface WebSocketMessage {
    type: string;
    payload: any;
}

interface WebSocketClientConfig {
    url: string;
    token: string;
    storeId: string;
}

export class WebSocketClient {
    private url: string;
    private token: string;
    private storeId: string;
    private ws: WebSocket | null = null;
    private state: ConnectionState = 'disconnected';
    private handlers: Map<string, Set<EventHandler>> = new Map();
    private reconnectAttempts = 0;
    private reconnectTimer: number | null = null;
    private pingInterval: number | null = null;
    private shouldReconnect = true;

    constructor(config: WebSocketClientConfig) {
        this.url = config.url;
        this.token = config.token;
        this.storeId = config.storeId;
    }

    connect(): Promise<void> {
        return new Promise((resolve, reject) => {
            if (this.ws?.readyState === WebSocket.OPEN) {
                resolve();
                return;
            }

            this.setState('connecting');

            // Build WebSocket URL - if relative (starts with /), prepend current origin
            let wsUrl: string;
            if (this.url.startsWith('/')) {
                const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
                wsUrl = `${protocol}//${window.location.host}${this.url}`;
            } else {
                wsUrl = this.url;
            }

            // Build query parameters: store_id always, token only if provided
            const params = new URLSearchParams();
            params.set('store_id', this.storeId);
            if (this.token) {
                params.set('token', this.token);
            }
            const queryString = params.toString();
            if (queryString) {
                wsUrl += '?' + queryString;
            }

            console.log('[WebSocket] Connecting to:', wsUrl);
            
            try {
                this.ws = new WebSocket(wsUrl);
            } catch (err) {
                console.error('[WebSocket] Connection failed:', err);
                this.setState('disconnected');
                reject(err);
                return;
            }

            this.ws.onopen = () => {
                console.log('[WebSocket] Connected successfully');
                this.reconnectAttempts = 0;
                this.setState('connected');
                this.startHeartbeat();
                resolve();
            };

            this.ws.onclose = (event) => {
                console.log('[WebSocket] Closed:', event.code, event.reason);
                this.stopHeartbeat();
                this.setState('disconnected');
                
                if (event.code === 1008) {
                    this.emit('auth_error', { reason: event.reason });
                } else if (this.shouldReconnect) {
                    this.scheduleReconnect();
                }
            };

            this.ws.onerror = (error) => {
                console.error('[WebSocket] Error:', error);
                this.emit('error', error);
            };

            this.ws.onmessage = (event) => {
                this.handleMessage(event);
            };
        });
    }

    disconnect(): void {
        this.shouldReconnect = false;
        this.stopHeartbeat();
        
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        
        if (this.ws) {
            this.ws.close(1000, 'Client disconnecting');
            this.ws = null;
        }
    }

    send(message: any): void {
        if (this.ws?.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        } else {
            console.warn('[WebSocket] Cannot send - not connected');
        }
    }

    on(event: string, handler: EventHandler): void {
        if (!this.handlers.has(event)) {
            this.handlers.set(event, new Set());
        }
        this.handlers.get(event)!.add(handler);
    }

    off(event: string, handler: EventHandler): void {
        const eventHandlers = this.handlers.get(event);
        if (eventHandlers) {
            eventHandlers.delete(handler);
        }
    }

    offAll(): void {
        this.handlers.clear();
    }

    getState(): ConnectionState {
        return this.state;
    }

    getReconnectAttempts(): number {
        return this.reconnectAttempts;
    }

    private setState(state: ConnectionState): void {
        this.state = state;
        this.emit('state_change', state);
    }

    private scheduleReconnect(): void {
        if (!this.shouldReconnect) return;

        const baseDelay = Math.pow(2, Math.min(this.reconnectAttempts, 5)) * 1000;
        const jitter = Math.random() * 200;
        const delay = Math.min(30000, baseDelay + jitter);

        this.reconnectAttempts++;
        this.setState('reconnecting');

        this.reconnectTimer = window.setTimeout(async () => {
            try {
                await this.connect();
                if (this.reconnectAttempts > 0) {
                    this.emit('reconnected', { stale: true });
                }
            } catch (err) {
                console.error('[WebSocket] Reconnect failed:', err);
            }
        }, delay);
    }

    private startHeartbeat(): void {
        this.stopHeartbeat();
        this.pingInterval = window.setInterval(() => {
            if (this.ws?.readyState === WebSocket.OPEN) {
                this.send({ type: 'ping' });
            }
        }, 54000);
    }

    private stopHeartbeat(): void {
        if (this.pingInterval) {
            clearInterval(this.pingInterval);
            this.pingInterval = null;
        }
    }

    private handleMessage(event: MessageEvent): void {
        try {
            const message: WebSocketMessage = JSON.parse(event.data);
            
            if (message.type === 'pong') {
                return;
            }

            this.emit(message.type, message.payload);
        } catch (err) {
            console.error('[WebSocket] Failed to parse message:', err);
        }
    }

    private emit(event: string, data: any): void {
        const eventHandlers = this.handlers.get(event);
        if (eventHandlers) {
            eventHandlers.forEach(handler => {
                try {
                    handler(data);
                } catch (err) {
                    console.error(`[WebSocket] Handler error for event "${event}":`, err);
                }
            });
        }
    }
}
