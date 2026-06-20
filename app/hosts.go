package app

import (
	"log/slog"

	"github.com/drydock/drydock/internal/core/domain"
)

// HostDTO is the typed host shape delivered to the frontend.
type HostDTO struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Transport     string `json:"transport"`
	Endpoint      string `json:"endpoint"`
	Trust         string `json:"trust"`
	ObserveMode   bool   `json:"observeMode"`
	Connected     bool   `json:"connected"`
	EngineVersion string `json:"engineVersion"`
	APIVersion    string `json:"apiVersion"`
}

// AddHostInput is the payload from the add-host wizard.
type AddHostInput struct {
	Name        string `json:"name"`
	Transport   string `json:"transport"`
	Endpoint    string `json:"endpoint"`
	ObserveMode bool   `json:"observeMode"`
}

// ListHosts returns all known host profiles with their connection state.
func (a *App) ListHosts() []HostDTO {
	statuses := a.registry.List()
	out := make([]HostDTO, 0, len(statuses))
	for _, s := range statuses {
		out = append(out, toHostDTO(s.Host, s.Connected))
	}
	return out
}

// AddHost saves a new host profile and attempts to connect it. The profile is
// returned regardless of whether the connection succeeded, so the UI can show a
// disconnected host with a remediation hint.
func (a *App) AddHost(input AddHostInput) (HostDTO, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()

	host, err := a.registry.Add(ctx, domain.Host{
		Name:        input.Name,
		Transport:   domain.Transport(input.Transport),
		Endpoint:    input.Endpoint,
		ObserveMode: input.ObserveMode,
	})
	if err != nil {
		return HostDTO{}, err
	}

	connected, connErr := a.registry.Connect(ctx, host.ID)
	if connErr != nil {
		a.log.Info("host added but not connected", slog.String("host", host.Name), slog.Any("error", connErr))
		return toHostDTO(host, false), nil
	}
	return toHostDTO(connected, true), nil
}

// ConnectHost connects a saved host.
func (a *App) ConnectHost(id string) (HostDTO, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	host, err := a.registry.Connect(ctx, id)
	if err != nil {
		return HostDTO{}, err
	}
	return toHostDTO(host, true), nil
}

// DisconnectHost disconnects a host without removing its profile.
func (a *App) DisconnectHost(id string) error {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.registry.Disconnect(ctx, id)
}

// RemoveHost disconnects and deletes a host profile.
func (a *App) RemoveHost(id string) error {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.registry.Remove(ctx, id)
}

// SetObserveMode toggles observe-only for a host (explicit, audited; ADR-0013).
func (a *App) SetObserveMode(id string, observe bool) error {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.registry.SetObserveMode(ctx, id, observe)
}

func toHostDTO(h domain.Host, connected bool) HostDTO {
	return HostDTO{
		ID:            h.ID,
		Name:          h.Name,
		Transport:     string(h.Transport),
		Endpoint:      h.Endpoint,
		Trust:         string(h.Trust),
		ObserveMode:   h.ObserveMode,
		Connected:     connected,
		EngineVersion: h.EngineVersion,
		APIVersion:    h.APIVersion,
	}
}
