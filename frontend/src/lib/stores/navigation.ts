// Navigation (view) state, kept separate from runtime data stores
// (PROJECT-BOOK §7.11.10).

import { writable } from 'svelte/store';
import { ViewId } from '../types/view';

/** The active primary view. */
export const activeView = writable<ViewId>(ViewId.Containers);
