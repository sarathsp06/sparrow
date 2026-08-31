<script lang="ts">
  interface Props {
    status: string;
  }

  let { status }: Props = $props();

  // tone -> functional color token
  const config: Record<string, { tone: string; label: string; pulse?: boolean }> = {
    pending:  { tone: "warn", label: "Pending" },
    sending:  { tone: "beacon", label: "Sending", pulse: true },
    success:  { tone: "ok", label: "Success" },
    failed:   { tone: "bad", label: "Failed" },
    retrying: { tone: "warn", label: "Retrying", pulse: true },
    expired:  { tone: "idle", label: "Expired" },
  };

  const fallback = { tone: "idle", label: "Unknown", pulse: false };
  const c = $derived(config[status] ?? fallback);
</script>

<span
  class="chip tnum"
  style="color:var(--color-{c.tone});border-color:color-mix(in srgb,var(--color-{c.tone}) 35%,transparent);background:color-mix(in srgb,var(--color-{c.tone}) 12%,var(--color-panel-2))"
>
  <span
    class="w-1.5 h-1.5 rounded-full {c.pulse ? 'animate-pulse' : ''}"
    style="background:var(--color-{c.tone})"
  ></span>
  {c.label}
</span>
