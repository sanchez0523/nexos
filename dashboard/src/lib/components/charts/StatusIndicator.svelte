<script lang="ts">
  export let online = false;
  export let label = '';
</script>

<div class="flex items-center gap-2 text-xs">
  <span
    class="inline-block w-2 h-2 rounded-full"
    class:bg-accent={online}
    class:bg-slate-600={!online}
  />
  <span class="text-slate-300">{label}</span>
  <span class="text-slate-500">{online ? 'online' : 'offline'}</span>
</div>
