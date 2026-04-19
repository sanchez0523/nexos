import type { AuthResponse } from '$lib/types';
import { api } from './client';

export function login(username: string, password: string) {
  return api<AuthResponse>('/api/auth/login', {
    method: 'POST',
    body: { username, password },
    skipAuthRefresh: true
  });
}

export function logout() {
  return api<void>('/api/auth/logout', { method: 'POST', skipAuthRefresh: true });
}

// Returns the refreshed AuthResponse (including username) or null when the
// refresh cookie is missing / expired. Callers use null as the "not authed"
// signal.
export async function refreshSession(): Promise<AuthResponse | null> {
  try {
    return await api<AuthResponse>('/api/auth/refresh', {
      method: 'POST',
      skipAuthRefresh: true
    });
  } catch {
    return null;
  }
}
