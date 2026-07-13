import { useWebSocket } from '$shared/api/websocket';

export function initWebSocket(): void {
  const token = sessionStorage.getItem('access_token') || '';
  useWebSocket().connect(token);
}
