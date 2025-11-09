# Generic Table Component Usage Examples

## Basic Usage

```svelte
<script>
    import { Table } from '$lib';
    
    let users = [
        { id: 1, name: 'John Doe', email: 'john@example.com', status: 'active' },
        { id: 2, name: 'Jane Smith', email: 'jane@example.com', status: 'inactive' }
    ];
</script>

<Table 
    headers={['id', 'name', 'email', 'status']} 
    data={users}
    itemName="users"
/>
```

## Advanced Usage with Column Formatters

```svelte
<script>
    import { Table } from '$lib';
    
    let webhooks = [
        { 
            id: 'wh_123', 
            name: 'User Events', 
            url: 'https://api.example.com/webhook',
            status: 'active',
            lastDelivery: 1699392000 
        }
    ];
</script>

{#snippet statusSnippet({ value })}
    <span 
        class="px-2 py-1 rounded-full text-xs font-medium"
        class:bg-green-100={value === 'active'}
        class:text-green-800={value === 'active'}
        class:bg-red-100={value === 'inactive'}
        class:text-red-800={value === 'inactive'}
    >
        {value}
    </span>
{/snippet}

{#snippet urlSnippet({ value })}
    <a href={value} class="text-blue-600 hover:text-blue-800 truncate" target="_blank">
        {value}
    </a>
{/snippet}

{#snippet lastDeliverySnippet({ value })}
    <time class="text-gray-600 text-sm">
        {value ? new Date(value * 1000).toLocaleString() : 'Never'}
    </time>
{/snippet}

{#snippet actionsSnippet({ row })}
    <div class="flex gap-2">
        <button 
            class="text-blue-600 hover:text-blue-800 text-sm font-medium"
            onclick={() => editWebhook(row.id)}
        >
            Edit
        </button>
        <button 
            class="text-red-600 hover:text-red-800 text-sm font-medium"
            onclick={() => deleteWebhook(row.id)}
        >
            Delete
        </button>
    </div>
{/snippet}

{#snippet emptyStateSnippet({ itemName })}
    <div class="text-center p-8">
        <h3 class="text-xl font-semibold mb-2">No webhooks configured</h3>
        <p class="text-gray-500 mb-4">Get started by creating your first webhook.</p>
        <button 
            class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700"
            onclick={() => createWebhook()}
        >
            Create Webhook
        </button>
    </div>
{/snippet}

<Table 
    headers={['name', 'url', 'status', 'lastDelivery']} 
    data={webhooks}
    itemName="webhooks"
    columnFormatters={{
        url: { header: 'Webhook URL', snippet: urlSnippet },
        status: { header: 'Status', snippet: statusSnippet },
        lastDelivery: { header: 'Last Delivery', snippet: lastDeliverySnippet }
    }}
    {actionsSnippet}
    {emptyStateSnippet}
    onRowClick={(row) => viewWebhookDetails(row.id)}
/>
```

## Component Props

### Required Props
- `headers: string[]` - Array of object keys to display as columns
- `data: Record<string, any>[]` - Array of objects (rows)

### Optional Props
- `itemName?: string` - Name for items (used in empty state, defaults to 'items')
- `columnFormatters?: Record<string, ColumnFormatter>` - Custom formatters for specific columns
- `error?: string` - Error message to display
- `loading?: boolean` - Loading state
- `emptyStateSnippet?: Snippet<[{ itemName: string }]>` - Custom empty state
- `actionsSnippet?: Snippet<[{ row: Record<string, any>; rowIndex: number }]>` - Actions column
- `onRowClick?: (row: Record<string, any>, rowIndex: number) => void` - Row click handler

### ColumnFormatter Interface
```typescript
interface ColumnFormatter {
    header?: string; // Custom header name, defaults to the key
    snippet?: Snippet<[{ value: any; row: Record<string, any>; rowIndex: number }]>;
}
```

## Benefits

✅ **Flexible Column Rendering**: Use snippets to customize any column  
✅ **Type Safety**: Full TypeScript support with proper interfaces  
✅ **Reusable**: One component for all table needs across the app  
✅ **Customizable**: Headers, actions, empty states all configurable  
✅ **Interactive**: Row clicks, hover effects, and custom actions  
✅ **Responsive**: Horizontal scroll on mobile devices  
✅ **Accessible**: Proper table semantics maintained  