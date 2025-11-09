<script lang="ts">
    import type { Snippet } from 'svelte';

    interface ColumnFormatter {
        header?: string; // Custom header name, defaults to the key
        snippet?: Snippet<[{ value: any; row: Record<string, any>; rowIndex: number }]>;
    }

    interface Props {
        itemName?: string;
        headers: string[]; // Array of object keys to display as columns
        data: Record<string, any>[]; // Array of objects (rows)
        columnFormatters?: Record<string, ColumnFormatter>; // Custom formatters for specific columns
        error?: string;
        loading?: boolean;
        emptyStateSnippet?: Snippet<[{ itemName: string }]>;
        actionsSnippet?: Snippet<[{ row: Record<string, any>; rowIndex: number }]>;
        onRowClick?: (row: Record<string, any>, rowIndex: number) => void;
    }

    let { 
        headers, 
        data, 
        columnFormatters = {},
        error, 
        loading, 
        itemName = 'items',
        emptyStateSnippet,
        actionsSnippet,
        onRowClick
    }: Props = $props();

    // Get display name for header
    function getHeaderName(key: string): string {
        return columnFormatters[key]?.header || key;
    }
</script>

{#if error}
    <div class="bg-red-100 border border-red-300 text-red-700 rounded-lg p-4 mb-6 flex items-center gap-3 shadow-sm">
        <span class="material-symbols-outlined">error</span>
        <p>{error}</p>
    </div>
{:else if loading}
    <div class="flex justify-center items-center h-40">
        <div class="animate-spin">
            <img src="/favicon.ico" alt="Loading..." class="w-8 h-8" />
        </div>
    </div>
{:else if data.length === 0}
    {#if emptyStateSnippet}
        {@render emptyStateSnippet({ itemName })}
    {:else}
        <div class="bg-white border rounded-lg p-8 text-center text-gray-500 shadow-sm flex flex-col items-center gap-4">
            <span class="material-symbols-outlined text-5xl text-gray-300">table_view</span>
            <h3 class="text-xl font-semibold">No {itemName} found</h3>
            <p>There are currently no {itemName} to display.</p>
        </div>
    {/if}
{:else}
    <!-- Generic Table -->
    <div class="bg-white rounded-lg shadow-sm border overflow-hidden">
        <div class="overflow-x-auto">
            <table class="w-full text-sm text-left">
                <thead class="text-xs text-gray-700 uppercase bg-gray-50">
                    <tr>
                        {#each headers as header}
                            <th class="px-4 py-3">
                                {getHeaderName(header)}
                            </th>
                        {/each}
                        {#if actionsSnippet}
                            <th class="px-4 py-3">Actions</th>
                        {/if}
                    </tr>
                </thead>
                <tbody>
                    {#each data as row, rowIndex}
                        <tr 
                            class="border-b hover:bg-gray-50"
                            class:cursor-pointer={onRowClick}
                            onclick={() => onRowClick?.(row, rowIndex)}
                        >
                            {#each headers as header}
                                <td class="px-4 py-3">
                                    {#if columnFormatters[header]?.snippet}
                                        {@render columnFormatters[header].snippet({ 
                                            value: row[header], 
                                            row, 
                                            rowIndex 
                                        })}
                                    {:else}
                                        {row[header] ?? 'N/A'}
                                    {/if}
                                </td>
                            {/each}
                            {#if actionsSnippet}
                                <td class="px-4 py-3">
                                    {@render actionsSnippet({ row, rowIndex })}
                                </td>
                            {/if}
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    </div>
{/if}

<style>
    .cursor-pointer {
        cursor: pointer;
    }
</style>