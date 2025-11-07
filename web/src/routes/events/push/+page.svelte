<script lang="ts">
  import {
    type Content,
    createAjvValidator,
    JSONEditor,
    Mode,
    type Validator
  } from "svelte-jsoneditor";

  import { client } from "$lib/services";
  import { create } from "@bufbuild/protobuf";
  import { onMount } from "svelte";
  import type {
    RegisteredEvent
  } from "../../../../../proto/webhook_pb.js";
  import { PushEventRequestSchema } from "../../../../../proto/webhook_pb.js";

  let namespace = $state("default");
  let event = $state("");
  let payload = $state({ json: {} } as Content);
  let loading = $state(false);
  let error = $state("");
  let successMessage = $state("");
  let availableEvents: RegisteredEvent[] = $state([]);

  function validator():Validator {
    const selectedEvent = availableEvents.find((e) => e.name === event);
	console.log("Selected Event:", selectedEvent?.schema);
    if (selectedEvent && selectedEvent.schema) {
      return createAjvValidator({ schema: JSON.parse(selectedEvent.schema) });
    }
    return createAjvValidator({ schema: {} });
  }

  function ValidatePayload(content: Content) {
    const validationResult = validator()(content);
    if (validationResult) {
      error= `Invalid payload: ${validationResult?.map((e) => e.message).join(", ")}`;
    }
	else {
	  error = "";
	}
  }

  async function fetchEvents() {
    try {
      const req = { activeOnly: true };
      const res = await client.listEvents(req);
      availableEvents = res.events || [];
      if (availableEvents.length > 0) {
        event = availableEvents[0].name;
      }
    } catch (e: any) {
      error = `Failed to load available events: ${e.message}`;
    }
  }

  onMount(fetchEvents);

  async function pushEvent(e : Event) {
	e.preventDefault();
    loading = true;
    error = "";
    successMessage = "";
    try {
      // Convert content to JSON object for protobuf Struct
      let payloadObj: any = {};
      if ("text" in payload && payload.text) {
        payloadObj = JSON.parse(payload.text);
      } else if ("json" in payload && payload.json !== undefined) {
        payloadObj = payload.json;
      }

      const req = create(PushEventRequestSchema, {
        namespace,
        event,
        payload: payloadObj,
      });
      const res = await client.pushEvent(req);
      successMessage = `Event pushed successfully! Event ID: ${res.eventId}`;
    } catch (e: any) {
      error = `Failed to push event: ${e.message}`;
    } finally {
      loading = false;
    }
  }
</script>

<div class="min-h-screen w-full bg-gray-50 font-display">
  <main class="w-full p-6 flex items-center justify-center">
    <div class="w-full bg-white rounded-lg shadow-sm border p-6 max-w-4xl">
      <h1 class="text-2xl font-bold text-gray-800 mb-4">Push a Test Event</h1>
      <form onsubmit={pushEvent} class="flex flex-col gap-4">
        <div>
          <label for="namespace" class="font-semibold text-gray-600"
            >Namespace</label
          >
          <input
            type="text"
            id="namespace"
            bind:value={namespace}
            class="w-full mt-1 p-2 border rounded-md"
            required
          />
        </div>
        <div>
          <label for="event" class="font-semibold text-gray-600">Event</label>
          <select
            id="event"
            bind:value={event}
            class="w-full mt-1 p-2 border rounded-md"
            required
          >
            {#each availableEvents as e}
              <option value={e.name}>{e.name}</option>
            {/each}
          </select>
        </div>
        <div>
          <label for="payload" class="font-semibold text-gray-600"
            >Payload (JSON)</label
          >
          <JSONEditor
		    validator={validator()}
            bind:content={payload}
            mode={Mode.text}
            mainMenuBar={false}
          />
        </div>
        <button
          type="submit"
          class="bg-primary text-white px-4 py-2 rounded-lg font-semibold hover:bg-primary/90 transition"
          disabled={loading}
        >
          {loading ? "Pushing..." : "Push Event"}
        </button>
      </form>
      {#if error}
        <div class="mt-4 bg-red-100 text-red-700 p-3 rounded-md">{error}</div>
      {/if}
      {#if successMessage}
        <div class="mt-4 bg-green-100 text-green-700 p-3 rounded-md">
          {successMessage}
        </div>
      {/if}
    </div>
  </main>
</div>

<style>
  .bg-primary {
    background-color: #1d4ed8;
  }
  .text-primary {
    color: #1d4ed8;
  }
</style>
