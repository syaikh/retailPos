import { useWebSocket } from '$shared/api/websocket';

export function initWebSocket(): void {
  const token = sessionStorage.getItem('access_token') || '';
  console.log('[WS Init] initWebSocket called, token length:', token.length, '| token prefix:', token.substring(0, 15));
  useWebSocket().connect(token);
}
