<script lang="ts">
  import ResourceView from './ResourceView.svelte';
  import { images } from '../stores/objects';
  import { formatBytes, shortId } from '../util/format';

  export let hostId: string;
</script>

<ResourceView
  {hostId}
  store={images}
  icon="images"
  emptyTitle="No images"
  emptyMessage="This host has no images pulled or built yet."
  let:rows
>
  <table class="dd-table">
    <thead>
      <tr>
        <th>Repository</th>
        <th>Tag</th>
        <th class="num">Size</th>
        <th>Status</th>
        <th>ID</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as img (img.ID)}
        <tr>
          <td>{img.Repo}</td>
          <td class="mono">{img.Tag}</td>
          <td class="num">{formatBytes(img.Size)}</td>
          <td class="muted">
            {#if img.Dangling}dangling{:else if img.InUse}in use{:else}unused{/if}
          </td>
          <td class="mono">{shortId(img.ID)}</td>
        </tr>
      {/each}
    </tbody>
  </table>
</ResourceView>
