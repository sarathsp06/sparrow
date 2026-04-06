<script lang="ts">
    /**
     * Minimal shape of the BatchJobStatus proto message.
     * We define it here to avoid import issues with protoc-gen-es type-only exports.
     */
    interface BatchStatus {
        status: string;
        total: number;
        processed: number;
        failed: number;
    }

    interface Props {
        /** Current batch job status object */
        batch: BatchStatus | undefined;
        /** Label for the operation, e.g. "Re-push" or "Retry" */
        label?: string;
        /** Called when user clicks Cancel */
        oncancel?: () => void;
        /** Called when batch reaches a terminal state */
        ondone?: () => void;
    }

    let { batch, label = 'Batch', oncancel, ondone }: Props = $props();

    let dismissed = $state(false);

    let progressPercent = $derived(
        batch && batch.total > 0
            ? Math.round(((batch.processed + batch.failed) / batch.total) * 100)
            : 0
    );

    let isTerminal = $derived(
        batch?.status === 'completed' || batch?.status === 'failed' || batch?.status === 'cancelled'
    );

    let statusColor = $derived.by(() => {
        if (!batch) return 'bg-gray-200';
        switch (batch.status) {
            case 'completed': return 'bg-green-500';
            case 'failed': return 'bg-red-500';
            case 'cancelled': return 'bg-gray-400';
            case 'processing': return 'bg-blue-500';
            default: return 'bg-yellow-500';
        }
    });

    let statusLabel = $derived.by(() => {
        if (!batch) return '';
        switch (batch.status) {
            case 'pending': return 'Queued';
            case 'processing': return 'Processing';
            case 'completed': return 'Completed';
            case 'failed': return 'Failed';
            case 'cancelled': return 'Cancelled';
            default: return batch.status;
        }
    });

    // Auto-dismiss after terminal state + delay
    let autoDismissTimer: ReturnType<typeof setTimeout> | undefined;

    $effect(() => {
        if (isTerminal && !dismissed) {
            ondone?.();
            autoDismissTimer = setTimeout(() => {
                dismissed = true;
            }, 5000);
        }
        return () => {
            if (autoDismissTimer) clearTimeout(autoDismissTimer);
        };
    });

    function dismiss() {
        dismissed = true;
    }
</script>

{#if batch && !dismissed}
    <div class="bg-white border border-gray-200 rounded-lg p-4 shadow-sm">
        <div class="flex items-center justify-between mb-2">
            <div class="flex items-center gap-2">
                <span class="text-sm font-semibold text-gray-900">{label}</span>
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-medium {
                    batch.status === 'completed' ? 'bg-green-50 text-green-700' :
                    batch.status === 'failed' ? 'bg-red-50 text-red-700' :
                    batch.status === 'cancelled' ? 'bg-gray-100 text-gray-500' :
                    batch.status === 'processing' ? 'bg-blue-50 text-blue-700' :
                    'bg-yellow-50 text-yellow-700'
                }">
                    {statusLabel}
                </span>
            </div>
            <div class="flex items-center gap-2">
                {#if !isTerminal && oncancel}
                    <button
                        onclick={oncancel}
                        class="px-2.5 py-1 text-xs font-medium text-red-700 bg-red-50 rounded-md hover:bg-red-100 border border-red-200 transition"
                    >
                        Cancel
                    </button>
                {/if}
                {#if isTerminal}
                    <button
                        onclick={dismiss}
                        class="p-1 text-gray-400 hover:text-gray-600 transition"
                        title="Dismiss"
                    >
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                {/if}
            </div>
        </div>

        <!-- Progress bar -->
        <div class="w-full bg-gray-100 rounded-full h-2 overflow-hidden mb-2">
            <div
                class="h-full rounded-full transition-all duration-500 {statusColor}"
                style="width: {progressPercent}%"
            ></div>
        </div>

        <!-- Counts -->
        <div class="flex items-center gap-4 text-xs text-gray-500">
            <span>{batch.processed + batch.failed} / {batch.total}</span>
            {#if batch.processed > 0}
                <span class="text-green-600">{batch.processed} succeeded</span>
            {/if}
            {#if batch.failed > 0}
                <span class="text-red-600">{batch.failed} failed</span>
            {/if}
            {#if !isTerminal && batch.status === 'processing'}
                <span class="flex items-center gap-1">
                    <span class="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse"></span>
                    In progress
                </span>
            {/if}
        </div>
    </div>
{/if}
