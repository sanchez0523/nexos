import type { Device, MetricsResponse, SensorList } from '$lib/types';
import { api } from './client';

export function listDevices() {
  return api<{ devices: Device[] }>('/api/devices');
}

export function listSensors(deviceID: string) {
  return api<SensorList>(`/api/devices/${encodeURIComponent(deviceID)}/sensors`);
}

export interface MetricsQuery {
  device_id: string;
  sensor: string;
  from: Date;
  to: Date;
  limit?: number;
}

export function queryMetrics(q: MetricsQuery) {
  const params = new URLSearchParams({
    device_id: q.device_id,
    sensor: q.sensor,
    from: q.from.toISOString(),
    to: q.to.toISOString()
  });
  if (q.limit !== undefined) params.set('limit', String(q.limit));
  return api<MetricsResponse>(`/api/metrics?${params.toString()}`);
}
