package app

import (
	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/provenance"
)

// Image provenance bindings (PROJECT-BOOK §7.12.5, ADR-0019). Listing is local
// only — it never calls a registry, so there is no background phone-home. The
// drift check is explicit and runs through the host's daemon.

// ListImageProvenance returns the local provenance of the host's tagged images:
// age, untagged/`:latest` status, and the running digest. No registry call.
func (a *App) ListImageProvenance(hostID string) ([]domain.ImageProvenance, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	eng, err := a.registry.Engine(hostID)
	if err != nil {
		return nil, err
	}
	images, err := eng.ListImages(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ImageProvenance, 0, len(images))
	for _, img := range images {
		if img.Dangling {
			continue // dangling images have no resolvable provenance
		}
		out = append(out, provenance.Assess(img))
	}
	return out, nil
}

// CheckImageDrift performs the explicit registry digest check for one image
// through the host's daemon (ADR-0019) and returns its updated provenance,
// including whether the tag has drifted from the running digest.
func (a *App) CheckImageDrift(hostID, imageRef string) (domain.ImageProvenance, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	eng, err := a.registry.Engine(hostID)
	if err != nil {
		return domain.ImageProvenance{}, err
	}

	images, err := eng.ListImages(ctx)
	if err != nil {
		return domain.ImageProvenance{}, err
	}
	var base domain.ImageProvenance
	for _, img := range images {
		if p := provenance.Assess(img); p.ImageRef == imageRef {
			base = p
			break
		}
	}
	base.HostRef = hostID

	digest, err := eng.RegistryDigest(ctx, imageRef)
	if err != nil {
		// Reflects the host's registry reachability, not the desktop's. Return the
		// local provenance unchecked so the UI can say "unreachable from host".
		return base, err
	}
	return provenance.WithRegistryDigest(base, digest), nil
}
