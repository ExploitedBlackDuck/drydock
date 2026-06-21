// Typed seam over the run/create bindings (PROJECT-BOOK §7.5, §2.8).
import { OptionCatalog, RunContainer } from '../../../wailsjs/go/app/App';
import type { app, domain } from '../../../wailsjs/go/models';

export type CatalogOption = app.OptionDTO;
export type RunSpec = domain.RunSpec;

/** The option catalog the builder renders from — no free-text flags (ADR-0011). */
export function optionCatalog(): Promise<CatalogOption[]> {
  return OptionCatalog();
}

/** Creates and starts a container from a builder-assembled spec; returns its id. */
export function runContainer(hostId: string, spec: RunSpec): Promise<string> {
  return RunContainer(hostId, spec);
}
