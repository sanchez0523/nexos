<script lang="ts">
  import { derived } from 'svelte/store';
  import Grid from '$lib/components/Grid.svelte';
  import SensorCard from '$lib/components/SensorCard.svelte';
  import StatusIndicator from '$lib/components/charts/StatusIndicator.svelte';
  import { deviceList, sensors, sensorKey } from '$lib/stores/devices';

  // Each grid item is keyed by "device|sensor". We derive the id list directly
  // from the live sensor set so Auto-Discovery is fully declarative.
  const cardIds = derived(sensors, ($s) => Array.from($s).sort());

  function parseID(id: string): { device: string; sensor: string } {
    const i = id.indexOf('|');
    return { device: id.slice(0, i), sensor: id.slice(i + 1) };
  }
</script>

<section class="mb-6">
  <h2 class="text-sm text-slate-400 font-medium mb-2">Devices</h2>
  {#if $deviceList.length === 0}
    <p class="text-slate-500 text-sm">
      No devices yet. Connect one to <code class="text-accent">devices/&lt;id&gt;/&lt;sensor&gt;</code> to
      start streaming.
    </p>
  {:else}
    <div class="flex flex-wrap gap-4">
      {#each $deviceList as d (d.device_id)}
        <StatusIndicator online={d.online} label={d.device_id} />
      {/each}
    </div>
  {/if}
</section>

<section>
  <h2 class="text-sm text-slate-400 font-medium mb-2">Live sensors</h2>
  {#if $cardIds.length === 0}
    <div class="text-slate-500 text-sm border border-border rounded-lg p-8 text-center">
      Waiting for the first metric…
    </div>
  {:else}
    <Grid ids={$cardIds} let:id>
      {@const parsed = parseID(id)}
      <SensorCard deviceID={parsed.device} sensor={parsed.sensor} />
    </Grid>
  {/if}
</section>
