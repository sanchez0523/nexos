import type { TimeRange } from '$lib/types';

export function rangeToMs(r: TimeRange): number {
  switch (r) {
    case '15m':
      return 15 * 60 * 1000;
    case '1h':
      return 60 * 60 * 1000;
    case '6h':
      return 6 * 60 * 60 * 1000;
    case '24h':
      return 24 * 60 * 60 * 1000;
    case '7d':
      return 7 * 24 * 60 * 60 * 1000;
  }
}

// Ring buffer helper — keeps the most recent N points while metrics stream in.
export function appendCapped<T>(arr: T[], item: T, cap: number): T[] {
  if (arr.length >= cap) return [...arr.slice(arr.length - cap + 1), item];
  return [...arr, item];
}
