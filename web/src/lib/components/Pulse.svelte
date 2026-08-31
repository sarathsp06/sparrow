<script lang="ts">
  import { pulseStore } from "$lib/pulse.svelte";

  // The Pulse — Sparrow's signature. A live oscilloscope wire across the top of
  // the console. Blips encode the real delivery mix (success / pending / failed);
  // the beacon scan sweeps faster while the pipeline is working. Ambient idle line
  // when there's no traffic. Static under prefers-reduced-motion (see app.css).
  interface Props {
    height?: number;
  }
  let { height = 30 }: Props = $props();

  const MAX = 44;
  const toneVar: Record<string, string> = {
    ok: "var(--color-ok)",
    warn: "var(--color-warn)",
    bad: "var(--color-bad)",
    pending: "var(--color-beacon)",
  };

  // Deterministic-per-beat scatter so the wire "repaints" when telemetry updates.
  function scatter(seed: number): number {
    const x = Math.sin(seed * 12.9898) * 43758.5453;
    return x - Math.floor(x);
  }

  type Blip = { x: number; h: number; tone: string; pending: boolean };

  const blips = $derived.by<Blip[]>(() => {
    const total = pulseStore.total;
    const beat = pulseStore.beat; // dependency: rescatter on update
    if (total === 0) return [];
    const spec: [keyof typeof toneVar, number, number][] = [
      ["ok", pulseStore.ok, 4],
      ["warn", pulseStore.warn, 7],
      ["bad", pulseStore.bad, 11],
      ["pending", pulseStore.pending, 8],
    ];
    const out: Blip[] = [];
    let i = 0;
    for (const [tone, count, h] of spec) {
      const n = Math.max(count > 0 ? 1 : 0, Math.round((count / total) * MAX));
      for (let k = 0; k < n && out.length < MAX; k++) {
        out.push({ x: 20 + scatter(beat * 97 + i * 7.3 + k) * 1160, h, tone: toneVar[tone], pending: tone === "pending" });
        i++;
      }
    }
    return out;
  });

  const active = $derived(pulseStore.active);
</script>

<div class="pulse" class:active style="height:{height}px" aria-hidden="true">
  <svg class="wave" viewBox="0 0 1200 60" preserveAspectRatio="none">
    <line x1="0" y1="30" x2="1200" y2="30" class="baseline" />
    {#if blips.length === 0}
      <path class="trace" d="M0 30 H460 l10 -3 l8 6 H620 l6 -16 l7 30 l9 -26 l6 12 H960 l12 -4 l9 4 H1200" />
    {:else}
      {#each blips as b}
        <line class="blip" class:beacon={b.pending} x1={b.x} y1={30 - b.h} x2={b.x} y2={30 + b.h} style="stroke:{b.tone}" />
      {/each}
    {/if}
  </svg>
  <span class="scan"></span>
</div>

<style>
  .pulse {
    position: relative;
    width: 100%;
    overflow: hidden;
    mask-image: linear-gradient(90deg, transparent, #000 5%, #000 95%, transparent);
  }
  .wave { width: 100%; height: 100%; display: block; }
  .baseline {
    stroke: var(--color-line);
    stroke-width: 1;
    stroke-dasharray: 2 6;
    vector-effect: non-scaling-stroke;
  }
  .trace {
    fill: none;
    stroke: var(--color-beacon-dim);
    stroke-width: 1.5;
    opacity: 0.55;
    vector-effect: non-scaling-stroke;
    stroke-linejoin: round;
  }
  .blip {
    stroke-width: 1.5;
    vector-effect: non-scaling-stroke;
    opacity: 0.8;
  }
  .blip.beacon { animation: blink 1.6s ease-in-out infinite; }
  @keyframes blink { 0%, 100% { opacity: 0.35; } 50% { opacity: 1; } }
  .scan {
    position: absolute;
    top: 0;
    left: 0;
    height: 100%;
    width: 120px;
    pointer-events: none;
    background: linear-gradient(90deg, transparent, rgba(242, 169, 59, 0.06), rgba(242, 169, 59, 0.32));
    border-right: 1px solid var(--color-beacon);
    box-shadow: 0 0 12px rgba(242, 169, 59, 0.4);
    animation: sweep 7s linear infinite;
    will-change: transform;
  }
  .active .scan {
    animation-duration: 3.2s;
    box-shadow: 0 0 16px rgba(242, 169, 59, 0.65);
  }
  @keyframes sweep {
    from { transform: translateX(-120px); }
    to { transform: translateX(1240px); }
  }
</style>
