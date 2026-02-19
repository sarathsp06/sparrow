<script lang="ts">

  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import { eventClient as client } from "$lib/services";
  import { toJSONObject } from "$lib/utils";
  import { onMount } from "svelte";
  import { type JSONContent, JSONEditor, Mode } from "svelte-jsoneditor";

  let name = $state("");
  let description = $state("");
  let schema: JSONContent = $state({ json: {} });
  let active = $state(true);
  let error = $state("");

 

  $inspect(schema);
  onMount(async () => {
    const eventId = page.params.eventId;
    try {
      const req = { activeOnly: false };
      // TODO(sarath): expose get event endpoint
      const res = await client.listEvents(req);
      const event = res.events.find((e) => e.eventId === eventId);
      if (event) {
        name = event.name;
        description = event.description;
        schema = { json: event.schema || {} };
        active = event.active;
      }
    } catch (e: any) {
      error = `Failed to load event details: ${e.message}`;
    }
  });

  async function updateEvent(e: Event) {
    e.preventDefault();
    error = "";
    try {
      const req = {
        name,
        description,
        schema: toJSONObject(schema),
        active,
      };
      await client.updateEvent(req);
      goto("/events");
    } catch (e: any) {
      error = `Failed to update event: ${e.message}`;
    }
  }
</script>

<div
  class="min-h-screen bg-gradient-to-br from-white via-gray-50 to-gray-100 font-display"
>
  <div class="p-6 max-w-lg mx-auto">
    <header class="text-2xl font-bold mb-4 text-primary">Update Event</header>
    <form
      onsubmit={updateEvent}
      class="bg-white rounded-lg shadow p-6"
    >
      <div class="mb-4">
        <label for="name" class="block text-sm font-medium text-gray-700"
          >Name</label
        >
        <input
          type="text"
          id="name"
          bind:value={name}
          class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"
          disabled
        />
      </div>
      <div class="mb-4">
        <label for="description" class="block text-sm font-medium text-gray-700"
          >Description</label
        >
        <textarea
          id="description"
          bind:value={description}
          rows="3"
          class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"
        ></textarea>
      </div>
      <div class="mb-4">
        <label for="schema" class="block text-sm font-medium text-gray-700"
          >Schema (JSON)</label
        >
        			<JSONEditor bind:content={schema} mode={Mode.text} mainMenuBar={false}/>

      </div>
      <div class="mb-4">
        <label class="flex items-center">
          <input
            type="checkbox"
            bind:checked={active}
            class="rounded border-gray-300 text-indigo-600 shadow-sm focus:border-indigo-300 focus:ring focus:ring-offset-0 focus:ring-indigo-200 focus:ring-opacity-50"
          />
          <span class="ml-2 text-sm text-gray-600">Active</span>
        </label>
      </div>
      {#if error}
        <p class="text-red-500 text-sm mb-4">{error}</p>
      {/if}
      <button
        type="submit"
        class="w-full bg-primary text-white py-2 px-4 rounded-md hover:bg-primary-dark focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary"
        >Update</button
      >
    </form>
  </div>
</div>

<style>
  .bg-primary {
    background-color: #1d4ed8;
  }
  .text-primary {
    color: #1d4ed8;
  }
</style>
