<script lang="ts">
  interface Props {
    open: boolean;
    title?: string;
    message: string;
    confirmLabel?: string;
    cancelLabel?: string;
    variant?: "danger" | "warning" | "info";
    onconfirm: () => void;
    oncancel: () => void;
  }

  let {
    open,
    title = "Confirm",
    message,
    confirmLabel = "Confirm",
    cancelLabel = "Cancel",
    variant = "danger",
    onconfirm,
    oncancel,
  }: Props = $props();

  const tone = { danger: "bad", warning: "warn", info: "beacon" } as const;

  let confirmBtn = $state<HTMLButtonElement | null>(null);
  $effect(() => {
    if (open) confirmBtn?.focus();
  });
</script>

{#if open}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center p-4"
    role="dialog"
    aria-modal="true"
    aria-labelledby="confirm-title"
    aria-describedby="confirm-message"
    tabindex="-1"
    onkeydown={(e) => e.key === "Escape" && oncancel()}
  >
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="fixed inset-0 bg-ink/70 backdrop-blur-sm" role="presentation" onclick={oncancel}></div>
    <div class="panel ticked relative max-w-md w-full p-6">
      <span class="eyebrow" style="color:var(--color-{tone[variant]})">
        {variant === "danger" ? "Destructive action" : variant === "warning" ? "Caution" : "Confirm"}
      </span>
      <h3 id="confirm-title" class="text-lg font-semibold text-text mt-2 mb-2">{title}</h3>
      <p id="confirm-message" class="text-sm text-muted mb-6 leading-relaxed">{message}</p>
      <div class="flex justify-end gap-3">
        <button onclick={oncancel} class="btn btn-ghost">{cancelLabel}</button>
        <button
          bind:this={confirmBtn}
          onclick={onconfirm}
          class="btn {variant === 'danger' ? 'btn-danger' : 'btn-beacon'}"
        >
          {confirmLabel}
        </button>
      </div>
    </div>
  </div>
{/if}
