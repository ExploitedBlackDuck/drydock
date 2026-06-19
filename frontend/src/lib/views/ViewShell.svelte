<script lang="ts">
  // Shared frame for a primary view: a heading row with title + description and
  // a content region below.
  export let title: string;
  export let description = '';
</script>

<section class="view">
  <div class="head">
    <h1>{title}</h1>
    {#if description}<p>{description}</p>{/if}
  </div>
  <div class="body">
    <slot />
  </div>
</section>

<style>
  .view {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .head {
    flex: none;
    padding: var(--space-5) var(--space-5) var(--space-4);
    border-bottom: 1px solid var(--color-border);
  }

  h1 {
    margin: 0;
    font-size: var(--text-xl);
    font-weight: 650;
    letter-spacing: -0.01em;
  }

  p {
    margin: var(--space-1) 0 0;
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }

  .body {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }
</style>
