<script lang="ts">
  import ResourceView from './ResourceView.svelte';
  import { volumes } from '../stores/objects';
  import { formatBytes } from '../util/format';

  export let hostId: string;
</script>

<ResourceView
  {hostId}
  store={volumes}
  icon="volumes"
  emptyTitle="No volumes"
  emptyMessage="This host has no volumes. Each volume is confirmed individually before removal."
  let:rows
>
  <table class="dd-table">
    <thead>
      <tr>
        <th>Name</th>
        <th>Driver</th>
        <th class="num">Size</th>
        <th>Status</th>
        <th>Mountpoint</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as v (v.Name)}
        <tr>
          <td>{v.Name}</td>
          <td class="muted">{v.Driver}</td>
          <td class="num">{formatBytes(v.Size)}</td>
          <td class="muted">{v.InUse ? 'in use' : 'unused'}</td>
          <td class="mono">{v.Mountpoint}</td>
        </tr>
      {/each}
    </tbody>
  </table>
</ResourceView>
