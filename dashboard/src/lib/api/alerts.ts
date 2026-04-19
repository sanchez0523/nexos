import type { AlertRule, AlertRuleInput } from '$lib/types';
import { api } from './client';

export function listAlerts() {
  return api<{ alerts: AlertRule[] | null }>('/api/alerts');
}

export function createAlert(input: AlertRuleInput) {
  return api<AlertRule>('/api/alerts', { method: 'POST', body: input });
}

export function updateAlert(id: string, input: AlertRuleInput) {
  return api<AlertRule>(`/api/alerts/${id}`, { method: 'PUT', body: input });
}

export function deleteAlert(id: string) {
  return api<void>(`/api/alerts/${id}`, { method: 'DELETE' });
}
