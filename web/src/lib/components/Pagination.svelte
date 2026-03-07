<script lang="ts">
    interface Props {
        currentPage: number;
        totalPages: number;
        totalCount: number;
        pageSize: number;
        onPageChange: (pageNum: number) => void;
        itemLabel?: string;
    }

    let { currentPage, totalPages, totalCount, pageSize, onPageChange, itemLabel = 'items' }: Props = $props();

    function nextPage() {
        if (currentPage < totalPages) {
            onPageChange(currentPage + 1);
        }
    }

    

    function previousPage() {
        if (currentPage > 1) {
            onPageChange(currentPage - 1);
        }
    }
</script>

{#if totalPages > 1}
    <div class="mt-6 flex justify-between items-center">
        <div class="text-sm text-gray-500">
            Showing {((currentPage - 1) * pageSize) + 1} to {Math.min(currentPage * pageSize, totalCount)} of {totalCount} {itemLabel}
        </div>
        
        <div class="flex items-center gap-2">
            <button
                onclick={previousPage}
                disabled={currentPage === 1}
                class="px-3 py-2 text-sm font-medium text-gray-500 bg-white border border-gray-300 rounded-lg hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed"
            >
                Previous
            </button>
            
            <!-- Page Numbers -->
            <div class="flex items-center gap-1">
                {#each Array.from({ length: totalPages }, (_, i) => i + 1) as pageNum}
                    {#if pageNum === 1 || pageNum === totalPages || (pageNum >= currentPage - 2 && pageNum <= currentPage + 2)}
                        <button
                            onclick={() => onPageChange(pageNum)}
                            class={`px-3 py-2 text-sm font-medium rounded-lg ${
                                pageNum === currentPage
                                    ? 'bg-blue-600 text-white'
                                    : 'text-gray-500 bg-white border border-gray-300 hover:bg-gray-100'
                            }`}
                        >
                            {pageNum}
                        </button>
                    {:else if pageNum === currentPage - 3 || pageNum === currentPage + 3}
                        <span class="px-2 text-gray-400">...</span>
                    {/if}
                {/each}
            </div>
            
            <button
                onclick={nextPage}
                disabled={currentPage === totalPages}
                class="px-3 py-2 text-sm font-medium text-gray-500 bg-white border border-gray-300 rounded-lg hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed"
            >
                Next
            </button>
        </div>
    </div>
{/if}