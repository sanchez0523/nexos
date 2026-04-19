import { browser } from '$app/environment';
import { writable } from 'svelte/store';
import type { WsEvent } from '$lib/types';

// Connection lifecycle state. Only the most recent event is kept on the store
// — consumers that care about history subscribe and push to their own buffer.
export type WsStatus = 'idle' | 'connecting' | 'open' | 'closed';

export const wsStatus = writable<WsStatus>('idle');
export const lastEvent = writable<WsEvent | null>(null);

// Lightweight pub/sub so multiple stores (deviceStore, etc.) can react.
type Listener = (ev: WsEvent) => void;
const listeners = new Set<Listener>();

export function onWsEvent(l: Listener): () => void {
  listeners.add(l);
  return () => listeners.delete(l);
}

let socket: WebSocket | null = null;
let backoffMs = 1000;
const maxBackoffMs = 30_000;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let manualStop = false;

function wsURL(): string {
  const scheme = location.protocol === 'https:' ? 'wss' : 'ws';
  return `${scheme}://${location.host}/ws`;
}

function scheduleReconnect() {
  if (reconnectTimer !== null) return;
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    backoffMs = Math.min(backoffMs * 2, maxBackoffMs);
    connectWs();
  }, backoffMs);
}

export function connectWs() {
  if (!browser) return;
  if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) {
    return;
  }

  manualStop = false;
  wsStatus.set('connecting');
  const s = new WebSocket(wsURL());
  socket = s;

  s.addEventListener('open', () => {
    wsStatus.set('open');
    backoffMs = 1000; // reset on healthy connection
  });

  s.addEventListener('message', (msg) => {
    try {
      const ev = JSON.parse(msg.data) as WsEvent;
      lastEvent.set(ev);
      for (const l of listeners) l(ev);
    } catch {
      // drop malformed frames silently; the hub only emits JSON
    }
  });

  s.addEventListener('close', () => {
    wsStatus.set('closed');
    socket = null;
    if (!manualStop) scheduleReconnect();
  });

  s.addEventListener('error', () => {
    // `close` will fire after this; reconnect logic lives there.
    s.close();
  });
}

export function disconnectWs() {
  manualStop = true;
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (socket) {
    socket.close();
    socket = null;
  }
  wsStatus.set('idle');
}
