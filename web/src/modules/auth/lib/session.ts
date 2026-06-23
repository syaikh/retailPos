export function getAuthToken(): string | null {
  if (typeof window !== 'undefined') {
    return sessionStorage.getItem('access_token');
  }
  return null;
}

export function setAccessToken(token: string): void {
  sessionStorage.setItem('access_token', token);
}

export function removeAccessToken(): void {
  sessionStorage.removeItem('access_token');
}
