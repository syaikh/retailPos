import { writable } from 'svelte/store';

export const user = writable(null);
export const isAuthenticated = writable(false);

export const auth = {
  async checkAuth() {
    try {
      const response = await fetch('/api/validate', {
        credentials: 'include'
      });
      
      if (response.ok) {
        const data = await response.json();
        user.set(data.user);
        isAuthenticated.set(true);
        return true;
      }
    } catch (e) {
      console.error('Auth check failed:', e);
    }
    
    user.set(null);
    isAuthenticated.set(false);
    return false;
  },
  
  async logout() {
    await fetch('/api/logout', {
      method: 'POST',
      credentials: 'include'
    });
    user.set(null);
    isAuthenticated.set(false);
  }
};
