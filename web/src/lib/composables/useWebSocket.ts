import { onMount, onDestroy } from 'svelte';
import { writable } from 'svelte/store';
import { WebSocketClient } from '$lib/infrastructure/websocket';
import { goto } from '$app/navigation';
import { auth } from '$lib/stores/auth';
import { ui } from '$lib/stores/ui';
import type { ConnectionState } from '$lib/infrastructure/websocket';

interface UseWebSocketReturn {
    status: ReturnType<typeof writable<ConnectionState>>;
    lastError: ReturnType<typeof writable<string | null>>;
    setStockUpdateHandler: (fn: (payload: { product_id: number }) => void) => void;
    setSaleCreatedHandler: (fn: (payload: any) => void) => void;
    setReconnectedHandler: (fn: (payload: { stale: boolean }) => void) => void;
    disconnect: () => void;
}

export function useWebSocket(storeId: string): UseWebSocketReturn {
    const status = writable<ConnectionState>('disconnected');
    const lastError = writable<string | null>(null);
    
    let client: WebSocketClient | null = null;
    let onStockUpdate: ((payload: { product_id: number }) => void) | null = null;
    let onSaleCreated: ((payload: any) => void) | null = null;
    let onReconnected: ((payload: { stale: boolean }) => void) | null = null;
    
    // Debounce refetch helper
    let refetchTimer: number | null = null;
    
    // Track persistent connection error toast ID
    let connectionErrorToastId: number | null = null;

    const debouncedRefetch = (callback: () => void, delay = 300) => {
        if (refetchTimer) clearTimeout(refetchTimer);
        refetchTimer = window.setTimeout(callback, delay) as unknown as number;
    };

    const showConnectionErrorToast = () => {
        connectionErrorToastId = ui.warning('Tidak dapat terhubung ke server. Periksa koneksi internet.', 0);
    };

    const dismissConnectionErrorToast = () => {
        if (connectionErrorToastId !== null) {
            ui.removeToast(connectionErrorToastId);
            connectionErrorToastId = null;
        }
    };

    const connect = () => {
        // For same-origin WebSocket, authentication is handled via HTTP-only cookie
        // The browser automatically sends the session_token cookie with the upgrade request
        client = new WebSocketClient({
            url: '/api/ws',
            token: '', // unused; kept for compatibility
            storeId
        });

        // Event handlers
        client.on('state_change', (state) => {
            status.set(state);
            
            if (state === 'connected') {
                dismissConnectionErrorToast();
            } else if (state === 'reconnecting') {
                const attempts = client?.getReconnectAttempts() || 0;
                if (attempts >= 5 && connectionErrorToastId === null) {
                    showConnectionErrorToast();
                }
            }
        });

        client.on('stock.updated', (payload: { product_id: number }) => {
            const handler = onStockUpdate;
            if (handler) {
                debouncedRefetch(() => handler(payload), 300);
            }
        });

        client.on('sale.created', (payload: any) => {
            if (onSaleCreated) {
                onSaleCreated(payload);
            }
        });

        client.on('reconnected', (payload: { stale: boolean }) => {
            if (onReconnected) {
                onReconnected(payload);
            }
        });

        client.on('auth_error', () => {
            console.error('[WebSocket] Auth error - redirecting to login');
            auth.reset();
            localStorage.removeItem('token');
            sessionStorage.removeItem('refresh_token');
            goto('/login');
        });

        client.on('error', (err) => {
            lastError.set(err instanceof Error ? err.message : String(err));
        });

        client.connect().catch((err) => {
            console.error('[WebSocket] Connection failed:', err);
            lastError.set(err instanceof Error ? err.message : String(err));
        });
    };

    onMount(() => {
        connect();
    });

    onDestroy(() => {
        if (refetchTimer) {
            clearTimeout(refetchTimer);
        }
        client?.disconnect();
    });

    return {
        status,
        lastError,
        setStockUpdateHandler: (fn: (p: any) => void) => { onStockUpdate = fn; },
        setSaleCreatedHandler: (fn: (p: any) => void) => { onSaleCreated = fn; },
        setReconnectedHandler: (fn: (p: any) => void) => { onReconnected = fn; },
        disconnect: () => client?.disconnect()
    };
}
