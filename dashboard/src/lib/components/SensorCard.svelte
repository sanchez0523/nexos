<script lang="ts">
  import { onMount } from 'svelte';
  import LineChart from './charts/LineChart.svelte';
  import TimeRangePicker from './TimeRangePicker.svelte';
  import { onWsEvent } from '$lib/stores/ws';
  import { queryMetrics } from '$lib/api/devices';
  import type { MetricPoint, TimeRange } from '$lib/types';
  import { appendCapped, rangeToMs } from '$lib/utils';

  export let deviceID: string;
  export let sensor: string;

  let range: TimeRange = '1h';

  // Historical fetch on mount + on range change. Live events append to the
  // same array so the chart smoothly transitions from historical → real-time.
  const MAX_POINTS = 400;
  let points: MetricPoint[] = [];
  let latest: number | null = null;
  let loading = false;

  async function loadHistorical(r: TimeRange) {
    loading = true;
    try {
      const to = new Date();
      const from = new Date(to.getTime() - rangeToMs(r));
      const resp = await queryMetrics({ device_id: deviceID, sensor, from, to });
      points = resp.points ?? [];
      if (points.length > 0) latest = points[points.length - 1].value;
    } catch {
      // Non-fatal — keep stale data rather than blanking the card.
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    loadHistorical(range);
    return onWsEvent((ev) => {
      if (ev.device_id !== deviceID || ev.sensor !== sensor) return;
      latest = ev.value;
      points = appendCapped(points, { time: ev.time, value: ev.value }, MAX_POINTS);
    });
  });

  function onRangeChange(r: TimeRange) {
    range = r;
    loadHistorical(r);
  }
</script>

<div class="flex flex-col h-full p-3">
  <div class="flex items-center justify-between mb-2">
    <div class="flex flex-col">
      <span class="text-sm text-slate-100 font-medium">{sensor}</span>
      <span class="text-xs text-slate-500">{deviceID}</span>
    </div>
    <span class="text-sm tabular-nums text-slate-200">
      {latest !== null ? latest.toFixed(2) : '—'}
    </span>
  </div>

  <div class="flex-1 min-h-0">
    {#if loading && points.length === 0}
      <div class="h-full flex items-center justify-center text-slate-500 text-sm">Loading…</div>
    {:else}
      <LineChart {points} label={sensor} />
    {/if}
  </div>

  <div class="mt-2 flex items-center justify-end">
    <TimeRangePicker selected={range} onSelect={onRangeChange} />
  </div>
</div>
