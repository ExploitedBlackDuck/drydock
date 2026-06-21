package app

import (
	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/options"
)

// OptionDTO is a catalogued option exposed to the run/create builder. The UI
// renders inputs from these — there are no free-text flags (ADR-0011).
type OptionDTO struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Category    string `json:"category"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Risk        string `json:"risk"`
	AffectsData bool   `json:"affectsData"`
	Secret      bool   `json:"secret"`
}

// OptionCatalog returns the option catalog the run/create builder renders from.
func (a *App) OptionCatalog() ([]OptionDTO, error) {
	catalog, err := options.DefaultCatalog()
	if err != nil {
		return nil, err
	}
	opts := catalog.Options()
	out := make([]OptionDTO, 0, len(opts))
	for _, o := range opts {
		out = append(out, OptionDTO{
			Name:        o.Name,
			Type:        string(o.Type),
			Category:    o.Category,
			Summary:     o.Summary,
			Description: o.Description,
			Risk:        string(o.Risk),
			AffectsData: o.AffectsData,
			Secret:      o.Secret,
		})
	}
	return out, nil
}

// RunContainer creates and starts a container from a builder-assembled RunSpec
// (observe-aware, recorded, env redacted in capture). Returns the new id.
func (a *App) RunContainer(hostID string, spec domain.RunSpec) (string, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.ops.RunContainer(ctx, hostID, spec, true)
}
