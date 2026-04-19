import { derived, writable } from 'svelte/store';
import type { Device, WsEvent } from '$lib/types';
import { listDevices } from '$lib/api/devices';
import { onWsEvent } from './ws';

// Device registry keyed by device_id. Mutations come from two sources:
//   1. Initial REST fetch on dashboard load.
//   2. WebSocket events — each message either updates last_seen on a known
//      device or creates a new entry (Auto-Discovery).
//
// We also track the set of (device_id, sensor) pairs as a derived store, so
// the dashboard can add cards as new sensors appear in real time.

interface DeviceMap {
  [deviceID: string]: Device;
}

export const devices = writable<DeviceMap>({});

// Set<"device_id|sensor"> tracking every sensor ever observed this session.
// Used by the grid to decide when to create a new card.
export const sensors = writable<Set<string>>(new Set());

export const deviceList = derived(devices, ($d) => {
  return Object.values($d).sort((a, b) => b.last_seen.localeCompare(a.last_seen));
});

export function sensorKey(deviceID: string, sensor: string): string {
  return `${deviceID}|${sensor}`;
}

export async function refreshDevices(): Promise<void> {
  const { devices: list } = await listDevices();
  devices.update((m) => {
    const next: DeviceMap = {};
    for (const d of list) next[d.device_id] = d;
    return next;
  });
}

// Wire WS events into the device registry.
export function subscribeDeviceUpdates(): () => void {
  return onWsEvent((ev: WsEvent) => {
    devices.update((m) => {
      const existing = m[ev.device_id];
      m[ev.device_id] = {
        device_id: ev.device_id,
        first_seen: existing?.first_seen ?? ev.time,
        last_seen: ev.time,
        online: true
      };
      return m;
    });
    sensors.update((s) => {
      const k = sensorKey(ev.device_id, ev.sensor);
      if (!s.has(k)) {
        const next = new Set(s);
        next.add(k);
        return next;
      }
      return s;
    });
  });
}

// Recompute `online` flags client-side. Called on a ticker so stale devices
// flip to offline without waiting for a fresh REST fetch. Matches the backend
// convention (60s window).
const ONLINE_WINDOW_MS = 60_000;
export function recomputeOnline(now: Date = new Date()) {
  devices.update((m) => {
    const nowMs = now.getTime();
    for (const id of Object.keys(m)) {
      const seen = new Date(m[id].last_seen).getTime();
      m[id] = { ...m[id], online: nowMs - seen <= ONLINE_WINDOW_MS };
    }
    return m;
  });
}
