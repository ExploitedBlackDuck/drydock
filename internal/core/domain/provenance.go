package domain

import "time"

// ImageProvenance is the lightweight staleness signal for a running container's
// image (PROJECT-BOOK §7.12.5, ADR-0019): age, tag-vs-digest drift, and
// untagged/`:latest` ambiguity. The registry-digest fields are populated only by
// an explicit, operator-initiated check through the host's daemon — never a
// background phone-home, and credentials are the host's, referenced not copied.
type ImageProvenance struct {
	HostRef  string
	ImageRef string
	ImageID  string
	// RunningDigest is the digest the image was pulled as (from RepoDigests).
	RunningDigest string
	// RegistryDigest is the tag's current digest from the registry; empty until a
	// check runs.
	RegistryDigest string
	// TagDrifted is true when the running digest differs from the registry digest
	// (the tag moved since the image was pulled). Meaningful only once Checked.
	TagDrifted bool
	// Checked reports whether a registry digest check has been performed.
	Checked bool
	// Untagged is true for a dangling/`<none>` image (no resolvable provenance).
	Untagged bool
	// Latest is true when the tag is `:latest` — a moving, ambiguous reference.
	Latest  bool
	Created time.Time
}
