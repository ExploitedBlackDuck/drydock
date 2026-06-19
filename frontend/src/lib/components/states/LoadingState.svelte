<script lang="ts">
  // Streaming/loading indicator — non-blocking, per PROJECT-BOOK §7.11.9 (live
  // data with a streaming indicator, not a blocking spinner). Partial results
  // can render above it as they arrive.
  export let label = 'Loading…';
</script>

<div class="loading" role="status" aria-live="polite">
  <span class="bar" aria-hidden="true"></span>
  <span class="label">{label}</span>
</div>

<style>
  .loading {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-4);
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }

  .bar {
    position: relative;
    width: 120px;
    height: 3px;
    border-radius: 3px;
    background: var(--color-surface-raised);
    overflow: hidden;
  }

  .bar::after {
    content: '';
    position: absolute;
    inset: 0;
    width: 40%;
    border-radius: 3px;
    background: var(--color-accent);
    animation: slide 1.1s ease-in-out infinite;
  }

  @keyframes slide {
    0% {
      transform: translateX(-100%);
    }
    100% {
      transform: translateX(320%);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .bar::after {
      animation: none;
      width: 100%;
      opacity: 0.5;
    }
  }
</style>
