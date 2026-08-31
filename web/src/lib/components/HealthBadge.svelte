<script lang="ts">
  interface Props {
    health: string;
    size?: "sm" | "md";
  }

  let { health, size = "sm" }: Props = $props();

  const config: Record<string, { tone: string; label: string; tooltip?: string }> = {
    unknown:   { tone: "idle", label: "Idle", tooltip: "Health is calculated after the first deliveries" },
    healthy:   { tone: "ok", label: "Healthy" },
    degraded:  { tone: "warn", label: "Degraded" },
    unhealthy: { tone: "bad", label: "Unhealthy" },
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
