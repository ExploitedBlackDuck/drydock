// Package provenance computes a lightweight image-staleness signal (PROJECT-BOOK
// §7.12.5, ADR-0019): image age, untagged / `:latest` ambiguity, and tag-vs-
// digest drift. The local part is pure — no network — so that listing provenance
// never phones home; drift requires an explicit registry digest supplied by an
// operator-initiated check through the host's daemon. Vulnerability scanning is
// deliberately out of scope (a future Scanner port, never here).
package provenance

import (
	"github.com/drydock/drydock/internal/core/domain"
)

// Assess derives the local provenance of an image with no network call: its
// running digest, untagged / `:latest` status, and age. TagDrifted/Checked stay
// false until WithRegistryDigest is applied.
func Assess(img domain.Image) domain.ImageProvenance {
	ref := img.Repo
	if img.Tag != "" && img.Repo != "<none>" {
		ref = img.Repo + ":" + img.Tag
	}
	return domain.ImageProvenance{
		HostRef:       img.HostRef,
		ImageRef:      ref,
		ImageID:       img.ID,
		RunningDigest: img.RepoDigest,
		Untagged:      img.Dangling || img.Repo == "<none>" || img.Repo == "",
		Latest:        img.Tag == "latest",
		Created:       img.Created,
	}
}

// WithRegistryDigest records the result of an explicit registry check and
// computes drift. Drift is asserted only when both digests are known and differ
// — an unknown digest never produces a false "drifted".
func WithRegistryDigest(p domain.ImageProvenance, registryDigest string) domain.ImageProvenance {
	p.RegistryDigest = registryDigest
	p.Checked = true
	p.TagDrifted = p.RunningDigest != "" && registryDigest != "" && p.RunningDigest != registryDigest
	return p
}
