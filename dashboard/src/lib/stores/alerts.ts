import { writable } from 'svelte/store';
import type { AlertRule, AlertRuleInput } from '$lib/types';
import * as api from '$lib/api/alerts';

export const alerts = writable<AlertRule[]>([]);

export async function refreshAlerts() {
  const resp = await api.listAlerts();
  alerts.set(resp.alerts ?? []);
}

export async function createAlert(input: AlertRuleInput) {
  const created = await api.createAlert(input);
  alerts.update((a) => [created, ...a]);
  return created;
}

export async function updateAlert(id: string, input: AlertRuleInput) {
  const updated = await api.updateAlert(id, input);
  alerts.update((a) => a.map((r) => (r.id === id ? updated : r)));
  return updated;
}

export async function deleteAlert(id: string) {
  await api.deleteAlert(id);
  alerts.update((a) => a.filter((r) => r.id !== id));
}
