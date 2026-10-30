<!--
  A horizontal bar chart rendered at build time: plain HTML and CSS, no client
  JavaScript. Each value is printed beside its bar, so the numbers stay readable
  without the bars (and in the RSS feed).

    <BarChart
      title="Deliveries per second, one busy webhook"
      unit="/s"
      bars={[{ label: 'main', value: 193 }, { label: 'final', value: 881, highlight: true }]}
    />
-->
<script lang="ts">
  interface Bar {
    label: string;
    value: number;
    /** Draw this bar in the accent color. */
    highlight?: boolean;
  }

  interface Props {
    title: string;
    bars: Bar[];
    /** Appended to each printed value, e.g. "/s" or " ms". */
    unit?: string;
    /** Value of a full-width bar. Defaults to the largest value. */
    max?: number;
    /** A line under the chart: source, conditions, caveats. */
    caption?: string;
  }

  let { title, bars, unit = '', max, caption }: Props = $props();

  const scale = $derived(max ?? Math.max(...bars.map((b) => b.value)));
  const format = (v: number) => v.toLocaleString('en-US') + unit;
</script>

<figure class="fn-chart">
  <p class="fn-chart-title">{title}</p>
  <ul>
    {#each bars as bar}
      <li class:fn-chart-hl={bar.highlight}>
        <span class="fn-chart-label">{bar.label}</span>
        <span class="fn-chart-track" aria-hidden="true">
          <span class="fn-chart-bar" style:width="{(bar.value / scale) * 100}%"></span>
        </span>
        <span class="fn-chart-value">{format(bar.value)}</span>
      </li>
    {/each}
  </ul>
  {#if caption}<figcaption>{caption}</figcaption>{/if}
</figure>
