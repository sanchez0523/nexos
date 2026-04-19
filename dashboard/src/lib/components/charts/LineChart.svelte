<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import {
    Chart,
    LinearScale,
    PointElement,
    LineElement,
    LineController,
    TimeSeriesScale,
    Tooltip,
    Filler
  } from 'chart.js';

  Chart.register(
    LineController,
    LineElement,
    PointElement,
    LinearScale,
    TimeSeriesScale,
    Tooltip,
    Filler
  );

  // Time is plotted as a linear scale over epoch-ms — we keep the chart
  // self-contained by avoiding the date-fns adapter dependency.
  export let points: Array<{ time: string; value: number }> = [];
  export let label = '';

  let canvas: HTMLCanvasElement;
  let chart: Chart | null = null;

  function dataset() {
    return points.map((p) => ({ x: new Date(p.time).getTime(), y: p.value }));
  }

  onMount(() => {
    chart = new Chart(canvas, {
      type: 'line',
      data: {
        datasets: [
          {
            label,
            data: dataset(),
            borderColor: '#4ade80',
            backgroundColor: 'rgba(74,222,128,0.12)',
            borderWidth: 2,
            pointRadius: 0,
            tension: 0.25,
            fill: true
          }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        animation: false,
        parsing: false,
        scales: {
          x: {
            type: 'linear',
            ticks: {
              color: '#64748b',
              maxTicksLimit: 6,
              callback: (v) => new Date(v as number).toLocaleTimeString()
            },
            grid: { color: '#1e2630' }
          },
          y: {
            ticks: { color: '#64748b' },
            grid: { color: '#1e2630' }
          }
        },
        plugins: {
          legend: { display: false },
          tooltip: { mode: 'index', intersect: false }
        }
      }
    });
  });

  // Reactively push new data. We mutate dataset in place for liveliness and
  // avoid recreating the Chart instance on every frame.
  $: if (chart) {
    chart.data.datasets[0].data = dataset();
    chart.update('none');
  }

  onDestroy(() => {
    chart?.destroy();
    chart = null;
  });
</script>

<div class="relative w-full h-full">
  <canvas bind:this={canvas} />
</div>
