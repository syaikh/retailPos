// Auth API for checking and handling authentication
import { auth } from '$lib/stores/auth';

export async function checkAuth(): Promise<boolean> {
  try {
    const response = await fetch('/api/validate', {
      method: 'GET',
      credentials: 'include',
    });

    if (response.ok) {
      return true;
    } else {
      return false;
    }
  } catch (err) {
    return false;
  }
}

export async function login(username: string, password: string): Promise<boolean> {
  try {
    const response = await fetch('/api/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify({ username, password }),
    });

    if (response.ok) {
      const data = await response.json();
      // Store tokens for API access
      if (data.access_token) {
        sessionStorage.setItem('access_token', data.access_token);
      }
      if (data.refresh_token) {
        sessionStorage.setItem('refresh_token', data.refresh_token);
      }
      return true;
    } else {
      return false;
    }
  } catch (err) {
    return false;
  }
}

export async function logout(): Promise<void> {
  try {
    await fetch('/api/logout', {
      method: 'POST',
      credentials: 'include',
    });
  } catch (err) {
    // Ignore errors on logout
  }
  auth.clearUser();
}
