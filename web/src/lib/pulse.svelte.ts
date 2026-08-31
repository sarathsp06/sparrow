// Live telemetry backing the Pulse signature. The shell polls delivery stats
// into this store; the Pulse renders it as colored blips on the wire and
// speeds up while the pipeline is working (pending deliveries in flight).
import { browser, dev } from "$app/environment";

type Telemetry = { ok: number; warn: number; bad: number; pending: number };

let data = $state<Telemetry>({ ok: 0, warn: 0, bad: 0, pending: 0 });
// Bumped whenever a page reports fresh delivery activity, so the wire twitches.
let beat = $state(0);

export const pulseStore = {
  get ok() { return data.ok; },
  get warn() { return data.warn; },
  get bad() { return data.bad; },
  get pending() { return data.pending; },
  get total() { return data.ok + data.warn + data.bad + data.pending; },
  get active() { return data.pending > 0; },
  get beat() { return beat; },
  set(next: Telemetry) {
    data = next;
    beat++;
  },
  ping() {
    beat++;
  },
};

// Dev-only: expose on window so the signature can be exercised without a live
// delivery pipeline. Stripped from production builds.
if (dev && browser) {
  // window has no typed slot for our dev store handle
  const w = window as unknown as { __pulse?: typeof pulseStore };
  w.__pulse = pulseStore;
}
