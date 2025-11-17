<script lang="ts">

  import { goto } from '$app/navigation';
  import { client, JSONSchemaMetaSchema } from '$lib';
  import {
    createAjvValidator,
    JSONEditor,
    Mode,
    type JSONContent,
    type Validator
  } from "svelte-jsoneditor";
  
  let name = $state('');
  let description = $state('');
  let schema = $state({ json: {} } as JSONContent);
  let active = $state(true);
  let error = $state('');


  function validator():Validator {
      return createAjvValidator({ schema: JSONSchemaMetaSchema });
  }

  async function registerEvent(e: Event) {
    e.preventDefault();
    error = '';
    try {
      // Convert JSONContent to string for the API
      let schemaString: string = "{}";
      if ("text" in schema && typeof schema.text === "string") {
        schemaString = schema.text;
      } else if ("json" in schema && schema.json !== undefined) {
        schemaString = JSON.stringify(schema.json);
      }

      const req = {
        name,
        description,
        schema: schemaString,
        active,
      };
      await client.registerEvent(req);
      goto('/events');
    } catch (e) {
      error = 'Failed to register event';
    }
  }
</script>

<div class="min-h-screen bg-gradient-to-br from-white via-gray-50 to-gray-100 font-display">
  <div class="p-6 max-w-lg mx-auto">
    <h1 class="text-2xl font-bold mb-4">Register New Event</h1>
    <form onsubmit={registerEvent} class="bg-white rounded-lg shadow p-6">
      <div class="mb-4">
        <label for="name" class="block text-sm font-medium text-gray-700">Name</label>
        <input type="text" id="name" bind:value={name} class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
      </div>
      <div class="mb-4">
        <label for="description" class="block text-sm font-medium text-gray-700">Description</label>
        <textarea id="description" bind:value={description} rows="3" class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"></textarea>
      </div>
      <div class="mb-4">
        <label for="schema" class="block text-sm font-medium text-gray-700">Schema (JSON)</label>
        <JSONEditor  validator={validator()}  bind:content={schema} mode={Mode.text} mainMenuBar={false} />
      </div>
      <div class="mb-4">
        <label class="flex items-center">
          <input type="checkbox" bind:checked={active} class="rounded border-gray-300 text-indigo-600 shadow-sm focus:border-indigo-300 focus:ring focus:ring-offset-0 focus:ring-indigo-200 focus:ring-opacity-50">
          <span class="ml-2 text-sm text-gray-600">Active</span>
        </label>
      </div>
      {#if error}
        <p class="text-red-500 text-sm mb-4">{error}</p>
      {/if}
      <button type="submit" class="bg-blue-200 text-black py-2 px-4 rounded-md hover:bg-blue-300">Register</button>
    </form>
  </div>
</div>
