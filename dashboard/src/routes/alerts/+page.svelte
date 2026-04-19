<script lang="ts">
  import { onMount } from 'svelte';
  import type { AlertCondition, AlertRuleInput } from '$lib/types';
  import {
    alerts,
    refreshAlerts,
    createAlert,
    deleteAlert,
    updateAlert
  } from '$lib/stores/alerts';

  let showForm = false;
  let editingId: string | null = null;
  let form: AlertRuleInput = emptyForm();
  let error = '';

  function emptyForm(): AlertRuleInput {
    return {
      device_id: '',
      sensor: '',
      threshold: 0,
      condition: 'above',
      webhook_url: '',
      enabled: true
    };
  }

  onMount(() => {
    refreshAlerts().catch(() => {});
  });

  function openCreate() {
    editingId = null;
    form = emptyForm();
    showForm = true;
    error = '';
  }

  function openEdit(id: string) {
    const rule = $alerts.find((r) => r.id === id);
    if (!rule) return;
    editingId = id;
    form = {
      device_id: rule.device_id,
      sensor: rule.sensor,
      threshold: rule.threshold,
      condition: rule.condition,
      webhook_url: rule.webhook_url,
      enabled: rule.enabled
    };
    showForm = true;
    error = '';
  }

  async function onSubmit() {
    error = '';
    try {
      if (editingId) {
        await updateAlert(editingId, form);
      } else {
        await createAlert(form);
      }
      showForm = false;
    } catch (err) {
      error = 'Failed to save alert';
    }
  }

  async function onDelete(id: string) {
    if (!confirm('Delete this alert?')) return;
    try {
      await deleteAlert(id);
    } catch {
      // ignore — user can retry
    }
  }

  async function toggleEnabled(id: string) {
    const rule = $alerts.find((r) => r.id === id);
    if (!rule) return;
    try {
      await updateAlert(id, {
        device_id: rule.device_id,
        sensor: rule.sensor,
        threshold: rule.threshold,
        condition: rule.condition,
        webhook_url: rule.webhook_url,
        enabled: !rule.enabled
      });
    } catch {
      // ignore
    }
  }

  const conditions: AlertCondition[] = ['above', 'below'];
</script>

<div class="flex items-center justify-between mb-4">
  <h1 class="text-lg text-slate-100 font-semibold">Alert rules</h1>
  <button
    class="bg-accent text-black font-medium rounded-md px-3 py-1.5 text-sm"
    on:click={openCreate}
  >
    New alert
  </button>
</div>

{#if showForm}
  <form
    class="bg-panel border border-border rounded-xl p-4 mb-6 grid grid-cols-2 gap-3"
    on:submit|preventDefault={onSubmit}
  >
    <label class="text-xs text-slate-400 flex flex-col gap-1">
      Device ID
      <input
        type="text"
        required
        bind:value={form.device_id}
        class="bg-bg border border-border rounded px-2 py-1 text-slate-100"
      />
    </label>
    <label class="text-xs text-slate-400 flex flex-col gap-1">
      Sensor
      <input
        type="text"
        required
        bind:value={form.sensor}
        class="bg-bg border border-border rounded px-2 py-1 text-slate-100"
      />
    </label>
    <label class="text-xs text-slate-400 flex flex-col gap-1">
      Condition
      <select
        bind:value={form.condition}
        class="bg-bg border border-border rounded px-2 py-1 text-slate-100"
      >
        {#each conditions as c}
          <option value={c}>{c}</option>
        {/each}
      </select>
    </label>
    <label class="text-xs text-slate-400 flex flex-col gap-1">
      Threshold
      <input
        type="number"
        step="any"
        required
        bind:value={form.threshold}
        class="bg-bg border border-border rounded px-2 py-1 text-slate-100"
      />
    </label>
    <label class="text-xs text-slate-400 flex flex-col gap-1 col-span-2">
      Webhook URL
      <input
        type="url"
        required
        placeholder="https://example.com/hook"
        bind:value={form.webhook_url}
        class="bg-bg border border-border rounded px-2 py-1 text-slate-100"
      />
    </label>
    <label class="text-xs text-slate-300 flex items-center gap-2 col-span-2">
      <input type="checkbox" bind:checked={form.enabled} /> Enabled
    </label>
    {#if error}
      <p class="text-danger text-sm col-span-2">{error}</p>
    {/if}
    <div class="col-span-2 flex justify-end gap-2">
      <button type="button" class="text-xs text-slate-400" on:click={() => (showForm = false)}>
        Cancel
      </button>
      <button type="submit" class="bg-accent text-black rounded px-3 py-1.5 text-sm">
        {editingId ? 'Save changes' : 'Create alert'}
      </button>
    </div>
  </form>
{/if}

{#if $alerts.length === 0}
  <div class="text-slate-500 text-sm border border-border rounded-lg p-8 text-center">
    No alert rules yet.
  </div>
{:else}
  <table class="w-full text-sm">
    <thead>
      <tr class="text-left text-slate-500 text-xs border-b border-border">
        <th class="py-2">Device / Sensor</th>
        <th>Condition</th>
        <th>Threshold</th>
        <th>Webhook</th>
        <th>Enabled</th>
        <th></th>
      </tr>
    </thead>
    <tbody>
      {#each $alerts as rule (rule.id)}
        <tr class="border-b border-border/70 hover:bg-panel/60">
          <td class="py-2 text-slate-200">
            <div class="font-medium">{rule.device_id}</div>
            <div class="text-xs text-slate-500">{rule.sensor}</div>
          </td>
          <td class="text-slate-300">{rule.condition}</td>
          <td class="text-slate-300">{rule.threshold}</td>
          <td class="text-slate-400 truncate max-w-xs">{rule.webhook_url}</td>
          <td>
            <button
              class="text-xs px-2 py-0.5 rounded"
              class:bg-accent={rule.enabled}
              class:text-black={rule.enabled}
              class:bg-border={!rule.enabled}
              class:text-slate-400={!rule.enabled}
              on:click={() => toggleEnabled(rule.id)}
            >
              {rule.enabled ? 'enabled' : 'disabled'}
            </button>
          </td>
          <td class="text-right">
            <button class="text-xs text-slate-400 mr-3" on:click={() => openEdit(rule.id)}>
              Edit
            </button>
            <button class="text-xs text-danger" on:click={() => onDelete(rule.id)}>Delete</button>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}
