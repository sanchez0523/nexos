<script lang="ts">
  import { onDestroy, onMount, tick } from 'svelte';
  import { GridStack, type GridStackNode } from 'gridstack';

  // Items are addressed by stable IDs so we can persist layout to localStorage.
  // The consumer passes `ids` and renders each via the `item` slot, and we
  // rely on gridstack's `makeWidget` on existing DOM rather than its
  // programmatic addWidget — this keeps Svelte as the source of truth for the
  // actual content.
  export let ids: string[] = [];
  export let storageKey = 'nexos.grid.layout.v1';

  let container: HTMLDivElement;
  let grid: GridStack | null = null;

  interface Saved {
    [id: string]: { x: number; y: number; w: number; h: number };
  }

  function loadSaved(): Saved {
    if (typeof localStorage === 'undefined') return {};
    try {
      return JSON.parse(localStorage.getItem(storageKey) ?? '{}') as Saved;
    } catch {
      return {};
    }
  }

  function saveLayout() {
    if (!grid) return;
    const out: Saved = {};
    grid.getGridItems().forEach((el) => {
      const node = (el as unknown as { gridstackNode?: GridStackNode }).gridstackNode;
      const id = el.getAttribute('data-id');
      if (id && node) {
        out[id] = {
          x: node.x ?? 0,
          y: node.y ?? 0,
          w: node.w ?? 4,
          h: node.h ?? 3
        };
      }
    });
    localStorage.setItem(storageKey, JSON.stringify(out));
  }

  onMount(() => {
    // Apply any saved positions to the existing DOM nodes BEFORE init so
    // GridStack picks them up as it processes .grid-stack-item children.
    const saved = loadSaved();
    for (const el of Array.from(container.querySelectorAll<HTMLElement>('.grid-stack-item'))) {
      const id = el.getAttribute('data-id');
      if (!id) continue;
      const s = saved[id];
      if (s) {
        el.setAttribute('gs-x', String(s.x));
        el.setAttribute('gs-y', String(s.y));
        el.setAttribute('gs-w', String(s.w));
        el.setAttribute('gs-h', String(s.h));
      }
    }

    grid = GridStack.init(
      {
        column: 12,
        cellHeight: 90,
        margin: 10,
        float: true,
        animate: true
      },
      container
    );
    grid.on('change', saveLayout);
  });

  // Whenever `ids` changes, register any new item with gridstack applying
  // the saved layout (if any), otherwise letting gridstack auto-place it.
  $: (async () => {
    if (!grid) return;
    await tick(); // wait for Svelte to paint new children
    const saved = loadSaved();
    const registered = new Set(
      grid.getGridItems().map((el) => el.getAttribute('data-id') ?? '')
    );
    for (const id of ids) {
      if (registered.has(id)) continue;
      const el = container.querySelector<HTMLElement>(`[data-id="${CSS.escape(id)}"]`);
      if (!el) continue;
      const s = saved[id];
      if (s) {
        el.setAttribute('gs-x', String(s.x));
        el.setAttribute('gs-y', String(s.y));
        el.setAttribute('gs-w', String(s.w));
        el.setAttribute('gs-h', String(s.h));
      } else {
        // Default card size for a freshly discovered sensor.
        el.setAttribute('gs-w', '4');
        el.setAttribute('gs-h', '3');
      }
      grid.makeWidget(el);
    }
  })();

  onDestroy(() => {
    grid?.destroy(false);
    grid = null;
  });
</script>

<div bind:this={container} class="grid-stack">
  {#each ids as id (id)}
    <div class="grid-stack-item" data-id={id} gs-w="4" gs-h="3">
      <div class="grid-stack-item-content">
        <slot {id} />
      </div>
    </div>
  {/each}
</div>
