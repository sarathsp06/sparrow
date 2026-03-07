<script lang="ts">
  interface Props {
    open: boolean;
    title?: string;
    message: string;
    confirmLabel?: string;
    cancelLabel?: string;
    variant?: 'danger' | 'warning' | 'info';
    onconfirm: () => void;
    oncancel: () => void;
  }

  let {
    open,
    title = 'Confirm',
    message,
    confirmLabel = 'Confirm',
    cancelLabel = 'Cancel',
    variant = 'danger',
    onconfirm,
    oncancel,
  }: Props = $props();

  const variantStyles = {
    danger:  { btn: 'bg-red-600 hover:bg-red-700 focus:ring-red-500', icon: 'text-red-600' },
    warning: { btn: 'bg-yellow-600 hover:bg-yellow-700 focus:ring-yellow-500', icon: 'text-yellow-600' },
    info:    { btn: 'bg-blue-600 hover:bg-blue-700 focus:ring-blue-500', icon: 'text-blue-600' },
  };
</script>

{#if open}
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onkeydown={(e) => e.key === 'Escape' && oncancel()}
  >
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="fixed inset-0 bg-black/40 backdrop-blur-sm" role="presentation" onclick={oncancel}></div>
    <div class="relative bg-white rounded-xl shadow-2xl max-w-md w-full mx-4 p-6">
      <h3 class="text-lg font-semibold text-gray-900 mb-2">{title}</h3>
      <p class="text-sm text-gray-600 mb-6">{message}</p>
      <div class="flex justify-end gap-3">
        <button
          onclick={oncancel}
          class="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition"
        >
          {cancelLabel}
        </button>
        <button
          onclick={onconfirm}
          class="px-4 py-2 text-sm font-medium text-white rounded-lg transition focus:outline-none focus:ring-2 focus:ring-offset-2 {variantStyles[variant].btn}"
        >
          {confirmLabel}
        </button>
      </div>
    </div>
  </div>
{/if}
