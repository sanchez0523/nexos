// TypeScript mirrors of the Go JSON surfaces. Keep in lock-step with the
// structs in ingestion/internal/api and ingestion/internal/db.

export interface Device {
  device_id: string;
  first_seen: string; // ISO-8601
  last_seen: string;  // ISO-8601
  online: boolean;
}

export interface SensorList {
  device_id: string;
  sensors: string[];
}

export interface MetricPoint {
  time: string; // ISO-8601
  value: number;
}

export interface MetricsResponse {
  device_id: string;
  sensor: string;
  bucketed: boolean;
  points: MetricPoint[];
}

export type AlertCondition = 'above' | 'below';

export interface AlertRule {
  id: string;
  device_id: string;
  sensor: string;
  threshold: number;
  condition: AlertCondition;
  webhook_url: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface AlertRuleInput {
  device_id: string;
  sensor: string;
  threshold: number;
  condition: AlertCondition;
  webhook_url: string;
  enabled: boolean;
}

// Real-time WebSocket event envelope emitted by the Go hub.
export interface WsEvent {
  device_id: string;
  sensor: string;
  value: number;
  time: string;
}

export interface AuthResponse {
  username: string;
  expires_in: number;
}

// Time-range presets shared between TimeRangePicker, SensorCard, and utils.
// Kept here (not in a .svelte file) so svelte-check can resolve the import
// without compiling the component.
export type TimeRange = '15m' | '1h' | '6h' | '24h' | '7d';
