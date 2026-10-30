<!--
  A short summary box at the top of a long post. Write the points as a Markdown
  list between the tags (MDX):

    <Tldr>
    - First point.
    - Second point.
    </Tldr>
-->
<script lang="ts">
  import type { Snippet } from 'svelte';

  let { children }: { children: Snippet } = $props();
</script>

<aside class="fn-tldr" aria-label="Summary">
  <p class="fn-tldr-title">TL;DR</p>
  {@render children()}
</aside>
