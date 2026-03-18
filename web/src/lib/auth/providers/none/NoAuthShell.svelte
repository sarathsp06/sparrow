<!--
  NoAuthShell — fallback auth shell when no auth provider is configured.

  Renders the layout without any sign-in UI or access gating.
  This is the default mode for local development and simple deployments.
-->
<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    header,
    children,
    isPublicRoute = false,
  }: {
    header: Snippet;
    children: Snippet;
    isPublicRoute?: boolean;
  } = $props();
</script>

{#if !isPublicRoute}
  <header
    class="sticky flex w-full left-0 top-0 items-center justify-between px-8 py-2 z-999 bg-white/20 backdrop-blur-md border-b border-gray-100 shadow-xs"
  >
    {@render header()}
    <nav class="px-2 flex items-center flex-wrap gap-2 md:gap-8 text-lg font-medium">
      <a href="/webhooks" class="hover:text-primary transition">Webhooks</a>
      <a href="/events" class="hover:text-primary transition">Events</a>
      <a href="/health" class="hover:text-primary transition">Health</a>
      <a href="https://github.com/sarathsp06/sparrow" class="hover:text-primary transition">
        <img src="https://logo.svgcdn.com/devicon/github-original.png" alt="github" class="inline-block hover:text-primary w-8 h-8">
      </a>
    </nav>
  </header>
{/if}

{@render children()}
