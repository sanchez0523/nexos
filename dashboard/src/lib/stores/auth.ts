import { writable } from 'svelte/store';

export interface AuthState {
  username: string | null;
  checked: boolean; // has +layout.ts finished the initial refresh probe?
}

export const auth = writable<AuthState>({ username: null, checked: false });

export function setAuthed(username: string) {
  auth.set({ username, checked: true });
}

export function setUnauthed() {
  auth.set({ username: null, checked: true });
}
