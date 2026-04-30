/**
 * Authenticated fetch wrapper
 * Automatically attaches Bearer token from sessionStorage
 */
export async function apiFetch(url: string, options: RequestInit = {}): Promise<Response> {
  const token = sessionStorage.getItem('access_token');
  
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  };
  
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  
  return fetch(url, {
    ...options,
    headers,
  });
}
