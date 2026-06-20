<script lang="ts">
  import ResourceView from './ResourceView.svelte';
  import StateBadge from '../components/StateBadge.svelte';
  import { containers } from '../stores/objects';
  import type { Container } from '../api/engine';
  import { shortId } from '../util/format';

  export let hostId: string;

  function ports(c: Container): string {
    if (!c.Ports || c.Ports.length === 0) return '';
    return c.Ports.filter((p) => p.PublicPort)
      .map((p) => `${p.PublicPort}→${p.PrivatePort}/${p.Protocol}`)
      .join(', ');
  }
</script>

<ResourceView
  {hostId}
  store={containers}
  icon="containers"
  emptyTitle="No containers"
  emptyMessage="This host has no containers. They appear here as soon as one is created."
  let:rows
>
  <table class="dd-table">
    <thead>
      <tr>
        <th>Name</th>
        <th>State</th>
        <th>Image</th>
        <th>Ports</th>
        <th>Project</th>
        <th>ID</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as c (c.ID)}
        <tr>
          <td>{c.Name}</td>
          <td><StateBadge state={c.State} /></td>
          <td class="mono">{c.Image}</td>
          <td class="mono">{ports(c)}</td>
          <td class="muted">{c.ComposeProject}</td>
          <td class="mono">{shortId(c.ID)}</td>
        </tr>
      {/each}
    </tbody>
  </table>
</ResourceView>
