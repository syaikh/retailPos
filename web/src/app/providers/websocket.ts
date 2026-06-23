import { useWebSocket } from '$shared/api/websocket';

export function initWebSocket(): void {
  const token = sessionStorage.getItem('access_token');
  if (token && token.length > 10) {
    useWebSocket().connect(token);
  }
}
