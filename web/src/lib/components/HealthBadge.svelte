<script lang="ts">
  interface Props {
    health: string;
    size?: "sm" | "md";
  }

  let { health, size = "sm" }: Props = $props();

  const config: Record<string, { tone: string; label: string; tooltip?: string }> = {
    unknown:   { tone: "idle", label: "Idle", tooltip: "Health is calculated after the first deliveries" },
    healthy:   { tone: "ok", label: "Healthy", tooltip: "At least 90% of delivery attempts in the last 24 hours succeeded" },
    degraded:  { tone: "warn", label: "Degraded", tooltip: "Fewer than 90% of delivery attempts in the last 24 hours succeeded. Retries count, so a delivery that succeeded after failed attempts still lowers it" },
    unhealthy: { tone: "bad", label: "Unhealthy", tooltip: "Fewer than 80% of delivery attempts in the last 24 hours succeeded, or the last 5 attempts failed in a row" },
  };

  const fallback = config.unknown;
  const c = $derived(config[health] ?? fallback);

  const sizeCls = { sm: "px-2 py-0.5 text-[11px]", md: "px-2.5 py-1 text-xs" };
  const dotCls = { sm: "w-1.5 h-1.5", md: "w-2 h-2" };
</script>

<span
  class="inline-flex items-center gap-1.5 rounded-full mono {sizeCls[size]}"
  style="color:var(--color-{c.tone});border:1px solid color-mix(in srgb,var(--color-{c.tone}) 35%,transparent);background:color-mix(in srgb,var(--color-{c.tone}) 12%,var(--color-panel-2))"
  title={c.tooltip ?? ""}
>
  <span class="rounded-full {dotCls[size]}" style="background:var(--color-{c.tone})"></span>
  {c.label}
</span>
