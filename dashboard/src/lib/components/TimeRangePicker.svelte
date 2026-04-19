<script lang="ts">
  import type { TimeRange } from '$lib/types';

  export let selected: TimeRange = '1h';
  export let onSelect: (r: TimeRange) => void;

  const options: { value: TimeRange; label: string }[] = [
    { value: '15m', label: '15m' },
    { value: '1h', label: '1h' },
    { value: '6h', label: '6h' },
    { value: '24h', label: '24h' },
    { value: '7d', label: '7d' }
  ];
</script>

<div class="flex items-center gap-1 bg-bg rounded-md p-0.5 border border-border">
  {#each options as opt}
    <button
      type="button"
      class="px-2 py-0.5 text-xs rounded"
      class:bg-accent={selected === opt.value}
      class:text-black={selected === opt.value}
      class:text-slate-400={selected !== opt.value}
      on:click={() => {
        selected = opt.value;
        onSelect(opt.value);
      }}
    >
      {opt.label}
    </button>
  {/each}
</div>
