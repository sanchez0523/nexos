<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { auth, setAuthed, setUnauthed } from '$lib/stores/auth';
  import { refreshSession, logout } from '$lib/api/auth';
  import { connectWs, disconnectWs, wsStatus } from '$lib/stores/ws';
  import {
    refreshDevices,
    subscribeDeviceUpdates,
    recomputeOnline
  } from '$lib/stores/devices';

  let unsubscribeDeviceUpdates: (() => void) | null = null;
  let onlineTicker: ReturnType<typeof setInterval> | null = null;
  let devicesSeeded = false;

  // Single-flight auth probe on first mount. The API client also auto-refreshes
  // on 401; this just determines initial redirect.
  onMount(async () => {
    const session = await refreshSession();
    if (session) {
      setAuthed(session.username);
      // Seed the device list right after auth so the dashboard renders
      // known devices before the first WS event arrives.
      refreshDevices().catch(() => {});
    } else {
      setUnauthed();
    }
  });

  // React to auth changes: when authed, open WS + start tickers. When not,
  // redirect to /login (unless we're already there).
  $: if ($auth.checked) {
    if ($auth.username) {
      if (!unsubscribeDeviceUpdates) unsubscribeDeviceUpdates = subscribeDeviceUpdates();
      if (!onlineTicker) {
        onlineTicker = setInterval(() => recomputeOnline(), 15_000);
      }
      // Seed the device list once per authed session — covers both the
      // "refresh on initial mount succeeded" path and the "just logged in"
      // path. Subsequent updates come via WebSocket.
      if (!devicesSeeded) {
        devicesSeeded = true;
        refreshDevices().catch(() => {});
      }
      connectWs();
    } else {
      disconnectWs();
      if (unsubscribeDeviceUpdates) {
        unsubscribeDeviceUpdates();
        unsubscribeDeviceUpdates = null;
      }
      if (onlineTicker) {
        clearInterval(onlineTicker);
        onlineTicker = null;
      }
      devicesSeeded = false;
      if (!$page.url.pathname.startsWith('/login')) {
        goto('/login', { replaceState: true });
      }
    }
  }

  async function onLogout() {
    await logout();
    setUnauthed();
  }
</script>

{#if !$auth.checked}
  <div class="flex items-center justify-center h-full text-slate-400">Loading…</div>
{:else if $auth.username}
  <div class="flex flex-col h-full">
    <header class="flex items-center justify-between px-6 py-3 border-b border-border bg-panel">
      <a href="/" class="flex items-center gap-2 font-semibold text-accent">
        <span class="inline-block w-2 h-2 rounded-full bg-accent" />
        Nexos
      </a>
      <nav class="flex items-center gap-6 text-sm">
        <a href="/" class:underline={$page.url.pathname === '/'}>Dashboard</a>
        <a href="/alerts" class:underline={$page.url.pathname.startsWith('/alerts')}>Alerts</a>
        <span class="text-xs text-slate-500 capitalize">ws: {$wsStatus}</span>
        <button class="text-xs text-slate-400 hover:text-slate-100" on:click={onLogout}>
          Sign out
        </button>
      </nav>
    </header>
    <main class="flex-1 overflow-auto p-6">
      <slot />
    </main>
  </div>
{:else}
  <slot />
{/if}
