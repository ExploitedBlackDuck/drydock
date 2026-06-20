<script lang="ts">
  // A minimal inline-SVG sparkline. Pure presentation: give it a series and it
  // draws a normalized line.
  export let values: number[] = [];
  export let width = 160;
  export let height = 36;
  export let color = 'var(--color-accent)';

  $: points = (() => {
    if (values.length < 2) return '';
    const max = Math.max(...values, 0.0001);
    const min = Math.min(...values, 0);
    const span = max - min || 1;
    const step = width / (values.length - 1);
    return values
      .map((v, i) => {
        const x = i * step;
        const y = height - ((v - min) / span) * (height - 2) - 1;
        return `${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(' ');
  })();
</script>

<svg
  {width}
  {height}
  viewBox="0 0 {width} {height}"
  preserveAspectRatio="none"
  role="img"
  aria-label="trend"
>
  {#if points}
    <polyline
      {points}
      fill="none"
      stroke={color}
      stroke-width="1.5"
      stroke-linejoin="round"
      stroke-linecap="round"
    />
  {:else}
    <line
      x1="0"
      y1={height - 1}
      x2={width}
      y2={height - 1}
      stroke="var(--color-border-strong)"
      stroke-width="1"
    />
  {/if}
</svg>
