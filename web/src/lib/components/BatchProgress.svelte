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
        if (!batch) return 'bg-panel-2';
        switch (batch.status) {
            case 'completed': return 'bg-ok';
            case 'failed': return 'bg-bad';
            case 'cancelled': return 'bg-idle';
            case 'processing': return 'bg-beacon';
            default: return 'bg-warn';
        }
    });

    let statusTone = $derived.by(() => {
        if (!batch) return 'idle';
        switch (batch.status) {
            case 'completed': return 'ok';
            case 'failed': return 'bad';
            case 'cancelled': return 'idle';
            case 'processing': return 'beacon';
            default: return 'warn';
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
    <div class="panel p-4">
        <div class="flex items-center justify-between mb-2">
            <div class="flex items-center gap-2">
                <span class="text-sm font-semibold text-text">{label}</span>
                <span class="chip tnum" style="color:var(--color-{statusTone});border-color:color-mix(in srgb,var(--color-{statusTone}) 35%,transparent);background:color-mix(in srgb,var(--color-{statusTone}) 12%,var(--color-panel-2))">
                    {statusLabel}
                </span>
            </div>
            <div class="flex items-center gap-2">
                {#if !isTerminal && oncancel}
                    <button
                        onclick={oncancel}
                        class="btn btn-ghost !px-3 !py-1.5"
                    >
                        Cancel
                    </button>
                {/if}
                {#if isTerminal}
                    <button
                        onclick={dismiss}
                        class="p-1 text-faint hover:text-text transition"
                        title="Dismiss"
                        aria-label="Dismiss"
                    >
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                {/if}
            </div>
        </div>

        <!-- Progress bar -->
        <div class="w-full bg-panel-2 border border-line rounded-full h-2 overflow-hidden mb-2">
            <div
                class="h-full rounded-full transition-all duration-500 {statusColor}"
                style="width: {progressPercent}%"
            ></div>
        </div>

        <!-- Counts -->
        <div class="flex items-center gap-4 text-xs text-muted mono tnum">
            <span>{batch.processed + batch.failed} / {batch.total}</span>
            {#if batch.processed > 0}
                <span style="color:var(--color-ok)">{batch.processed} succeeded</span>
            {/if}
            {#if batch.failed > 0}
                <span style="color:var(--color-bad)">{batch.failed} failed</span>
            {/if}
            {#if !isTerminal && batch.status === 'processing'}
                <span class="flex items-center gap-1">
                    <span class="w-1.5 h-1.5 rounded-full bg-beacon animate-pulse"></span>
                    In progress
                </span>
            {/if}
        </div>
    </div>
{/if}
