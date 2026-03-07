<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    icon?: string;
    title: string;
    description?: string;
    action?: Snippet;
  }

  let { icon = 'inbox', title, description, action }: Props = $props();
</script>

<div class="flex flex-col items-center justify-center py-16 px-4 text-center">
  <div class="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
    <span class="material-symbols-outlined text-3xl text-gray-400">{icon}</span>
  </div>
  <h3 class="text-lg font-semibold text-gray-700 mb-1">{title}</h3>
  {#if description}
    <p class="text-sm text-gray-500 max-w-sm mb-4">{description}</p>
  {/if}
  {#if action}
    {@render action()}
  {/if}
</div>
