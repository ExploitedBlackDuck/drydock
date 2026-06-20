<script lang="ts">
  import ResourceView from './ResourceView.svelte';
  import { networks } from '../stores/objects';
  import { shortId } from '../util/format';

  export let hostId: string;
</script>

<ResourceView
  {hostId}
  store={networks}
  icon="networks"
  emptyTitle="No networks"
  emptyMessage="Only the default networks exist on this host."
  let:rows
>
  <table class="dd-table">
    <thead>
      <tr>
        <th>Name</th>
        <th>Driver</th>
        <th>Status</th>
        <th>ID</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as n (n.ID)}
        <tr>
          <td>{n.Name}</td>
          <td class="muted">{n.Driver}</td>
          <td class="muted">{n.InUse ? 'in use' : 'available'}</td>
          <td class="mono">{shortId(n.ID)}</td>
        </tr>
      {/each}
    </tbody>
  </table>
</ResourceView>
