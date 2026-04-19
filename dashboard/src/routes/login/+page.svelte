<script lang="ts">
  import { goto } from '$app/navigation';
  import { login } from '$lib/api/auth';
  import { setAuthed } from '$lib/stores/auth';

  let username = '';
  let password = '';
  let submitting = false;
  let error = '';

  async function onSubmit() {
    submitting = true;
    error = '';
    try {
      const resp = await login(username, password);
      setAuthed(resp.username);
      goto('/', { replaceState: true });
    } catch (err: unknown) {
      error = 'Invalid credentials';
    } finally {
      submitting = false;
    }
  }
</script>

<div class="flex items-center justify-center min-h-full py-16">
  <form
    class="w-full max-w-sm bg-panel border border-border rounded-xl p-8 space-y-4"
    on:submit|preventDefault={onSubmit}
  >
    <div class="flex items-center gap-2 text-accent font-semibold text-lg">
      <span class="inline-block w-2 h-2 rounded-full bg-accent" />
      Nexos
    </div>
    <h1 class="text-slate-200 text-xl font-semibold">Sign in</h1>

    <label class="block">
      <span class="text-xs text-slate-400">Username</span>
      <input
        class="mt-1 w-full bg-bg border border-border rounded-md px-3 py-2 text-slate-100 focus:outline-none focus:border-accent"
        type="text"
        autocomplete="username"
        bind:value={username}
        required
      />
    </label>

    <label class="block">
      <span class="text-xs text-slate-400">Password</span>
      <input
        class="mt-1 w-full bg-bg border border-border rounded-md px-3 py-2 text-slate-100 focus:outline-none focus:border-accent"
        type="password"
        autocomplete="current-password"
        bind:value={password}
        required
      />
    </label>

    {#if error}
      <p class="text-danger text-sm">{error}</p>
    {/if}

    <button
      type="submit"
      class="w-full py-2 rounded-md bg-accent text-black font-semibold disabled:opacity-60"
      disabled={submitting}
    >
      {submitting ? 'Signing in…' : 'Sign in'}
    </button>
  </form>
</div>
