// Static build: disable SSR entirely so the adapter emits a pure SPA bundle.
// Auth guard lives client-side in +layout.svelte.
export const ssr = false;
export const prerender = false;
