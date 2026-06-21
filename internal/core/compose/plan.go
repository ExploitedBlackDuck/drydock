package compose

import (
	"sort"

	"github.com/drydock/drydock/internal/core/domain"
)

// DesiredStack is the target state parsed from a project's Compose files (by the
// compose-go adapter, ADR-0016). It is a neutral input so the plan classifier
// below stays pure and table-testable, independent of the parser. A nil
// DesiredStack means the source was not accessible (source-unavailable).
type DesiredStack struct {
	Services []DesiredService
	Networks []string
	Volumes  []string
}

// DesiredService is a service as the Compose project declares it.
type DesiredService struct {
	Name string
	// ConfigHash is Compose's per-service config hash; "" when unknown.
	ConfigHash string
	// Image is the resolved image reference, used for the coarse diff when a
	// config hash cannot be compared with confidence.
	Image string
	// HasAnonymousVolumes is true when the service declares anonymous volumes,
	// which a recreate would drop.
	HasAnonymousVolumes bool
}

// ObservedStack is the running state read from the engine for a project.
type ObservedStack struct {
	Services []ObservedService
	Networks []string
	Volumes  []string
}

// ObservedService is a service's observed container state.
type ObservedService struct {
	Name    string
	Running bool
	// ConfigHash is the container's com.docker.compose.config-hash label; "" when
	// the label is absent and convergence cannot be confirmed.
	ConfigHash string
	Image      string
}

// sourceUnavailableNote labels a plan computed without the project source.
const sourceUnavailableNote = "source unavailable: the project's Compose files are not accessible to Drydock, so changes cannot be determined — showing observed state only"

// Plan classifies what applying a Compose project would do (ADR-0016, §7.12.2).
// A nil desired stack yields the explicit source-unavailable degraded plan: a
// labelled best-effort view of what is running, never a false "no changes". It
// is pure — no engine, no I/O, no shell — so it is table-tested against fixtures.
func Plan(project, hostRef string, desired *DesiredStack, observed ObservedStack) domain.ComposePlan {
	plan := domain.ComposePlan{Project: project, HostRef: hostRef}

	if desired == nil {
		return sourceUnavailablePlan(plan, observed)
	}

	observedByName := indexServices(observed.Services)
	for _, want := range desired.Services {
		change, degraded := classifyService(want, observedByName)
		if degraded {
			plan.Degraded = true
		}
		if change.InterruptsRunning || change.DropsAnonymousVolumes {
			plan.Destructive = true
		}
		plan.Services = append(plan.Services, change)
	}
	sort.Slice(plan.Services, func(i, j int) bool { return plan.Services[i].Service < plan.Services[j].Service })

	plan.Networks = createMissing(desired.Networks, observed.Networks, "network")
	plan.Volumes = createMissing(desired.Volumes, observed.Volumes, "volume")
	return plan
}

// classifyService decides one service's change, returning whether the decision
// was degraded (a config hash could not be compared with confidence).
func classifyService(want DesiredService, observed map[string]ObservedService) (domain.ServiceChange, bool) {
	change := domain.ServiceChange{Service: want.Name}

	have, exists := observed[want.Name]
	if !exists {
		change.Action = domain.ServiceCreate
		change.Reasons = []string{"no container exists for this service"}
		return change, false
	}

	// Confident comparison when both hashes are present.
	if want.ConfigHash != "" && have.ConfigHash != "" {
		if want.ConfigHash == have.ConfigHash {
			return unchanged(change, have), false
		}
		return recreate(change, want, have, "configuration changed"), false
	}

	// Degraded: fall back to a coarse image comparison, clearly labelled.
	if want.Image != "" && have.Image != "" && want.Image != have.Image {
		c := recreate(change, want, have, "coarse diff: image changed (config hash unavailable)")
		return c, true
	}
	change.Action = domain.ServiceNoop
	change.Reasons = []string{"coarse diff: no image change detected (config hash unavailable)"}
	return change, true
}

// unchanged returns a noop or start depending on whether the service is running.
func unchanged(change domain.ServiceChange, have ObservedService) domain.ServiceChange {
	if have.Running {
		change.Action = domain.ServiceNoop
		change.Reasons = []string{"up to date"}
	} else {
		change.Action = domain.ServiceStart
		change.Reasons = []string{"unchanged but stopped"}
	}
	return change
}

// recreate builds a recreate change, flagging the interrupt and anonymous-volume
// loss that make it destructive.
func recreate(change domain.ServiceChange, want DesiredService, have ObservedService, reason string) domain.ServiceChange {
	change.Action = domain.ServiceRecreate
	change.Reasons = []string{reason}
	change.InterruptsRunning = have.Running
	change.DropsAnonymousVolumes = want.HasAnonymousVolumes
	return change
}

// sourceUnavailablePlan builds the explicit degraded plan when the project
// source could not be read: every observed service is reported as
// "cannot determine", never a confident no-op (ADR-0016).
func sourceUnavailablePlan(plan domain.ComposePlan, observed ObservedStack) domain.ComposePlan {
	plan.Degraded = true
	plan.Notes = []string{sourceUnavailableNote}
	services := append([]ObservedService(nil), observed.Services...)
	sort.Slice(services, func(i, j int) bool { return services[i].Name < services[j].Name })
	for _, have := range services {
		plan.Services = append(plan.Services, domain.ServiceChange{
			Service: have.Name,
			Action:  domain.ServiceNoop,
			Reasons: []string{"source unavailable — change cannot be determined"},
		})
	}
	return plan
}

func indexServices(services []ObservedService) map[string]ObservedService {
	out := make(map[string]ObservedService, len(services))
	for _, s := range services {
		out[s.Name] = s
	}
	return out
}

// createMissing classifies named resources the project declares that are not yet
// present as create; existing ones are left alone (compose `up` never removes
// them).
func createMissing(desired, observed []string, kind string) []domain.ResourceChange {
	have := make(map[string]bool, len(observed))
	for _, name := range observed {
		have[name] = true
	}
	var changes []domain.ResourceChange
	for _, name := range desired {
		if !have[name] {
			changes = append(changes, domain.ResourceChange{Name: name, Kind: kind, Action: domain.ResourceCreate})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Name < changes[j].Name })
	return changes
}
