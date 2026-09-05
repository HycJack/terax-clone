import { usePreferencesStore } from "@/modules/settings/preferences";
import {
  setFavoriteModelIds,
  setRecentModelIds,
} from "@/modules/settings/store";

const RECENTS_MAX = 5;

// Serialize read-modify-write cycles. The store only reflects a write after
// its async IPC round-trip returns, so two rapid calls would otherwise both
// read the same stale snapshot and the second would clobber the first.
let prefWriteTail: Promise<unknown> = Promise.resolve();
function enqueuePrefWrite<T>(fn: () => Promise<T>): Promise<T> {
  const next = prefWriteTail.then(fn, fn);
  prefWriteTail = next.catch(() => undefined);
  return next;
}

export async function toggleFavoriteModel(id: string): Promise<void> {
  await enqueuePrefWrite(async () => {
    const current = usePreferencesStore.getState().favoriteModelIds;
    const next = current.includes(id)
      ? current.filter((x) => x !== id)
      : [...current, id];
    await setFavoriteModelIds(next);
  });
}

export async function pushRecentModel(id: string): Promise<void> {
  await enqueuePrefWrite(async () => {
    const current = usePreferencesStore.getState().recentModelIds;
    const next = [id, ...current.filter((x) => x !== id)].slice(0, RECENTS_MAX);
    await setRecentModelIds(next);
  });
}
