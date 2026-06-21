package domain

// StackState is a Compose stack's aggregate lifecycle state, derived from its
// containers (PROJECT-BOOK §7.11.6).
type StackState string

const (
	// StackRunning means every container in the stack is running.
	StackRunning StackState = "running"
	// StackPartial means some containers are running and some are not — the
	// classic "one service crashed" state the operator most needs to see.
	StackPartial StackState = "partial"
	// StackStopped means no container in the stack is running.
	StackStopped StackState = "stopped"
)

// Stack is a Compose project viewed as a single unit (PROJECT-BOOK §7.11.6): the
// containers Compose stamped with the same project label, grouped by service.
// HostRef is the host the stack was discovered on.
type Stack struct {
	Project  string
	HostRef  string
	Services []StackService
	// Running and Total count containers across the whole stack.
	Running int
	Total   int
	State   StackState
}

// StackService is one service within a stack and the containers backing it
// (a service may run more than one replica).
type StackService struct {
	Name       string
	Containers []Container
	Running    int
	Total      int
}

// ServiceAction is how a Compose service changes when the project is applied
// with `up` (PROJECT-BOOK §7.12.2, ADR-0016).
type ServiceAction string

const (
	// ServiceCreate creates a service that has no container yet.
	ServiceCreate ServiceAction = "create"
	// ServiceRecreate replaces a service whose configuration changed; this
	// interrupts a running container and may drop its anonymous volumes.
	ServiceRecreate ServiceAction = "recreate"
	// ServiceStart starts an existing, unchanged but stopped service.
	ServiceStart ServiceAction = "start"
	// ServiceNoop leaves a running, unchanged service as-is.
	ServiceNoop ServiceAction = "noop"
)

// ResourceAction is how a Compose network or volume changes on apply.
type ResourceAction string

const (
	// ResourceCreate creates a missing network or volume.
	ResourceCreate ResourceAction = "create"
	// ResourceRecreate replaces a changed network.
	ResourceRecreate ResourceAction = "recreate"
	// ResourceRemove removes a network or volume (destructive).
	ResourceRemove ResourceAction = "remove"
)

// ServiceChange is one service's classified change in a ComposePlan.
type ServiceChange struct {
	Service string
	Action  ServiceAction
	// Reasons explains the classification (e.g. "config changed", "source
	// unavailable — change cannot be determined"). A noop with reasons is a
	// degraded "cannot determine", never a confident "no change".
	Reasons []string
	// DropsAnonymousVolumes is true when a recreate would delete the service's
	// anonymous volumes — persistent data is lost (routes to require_ack).
	DropsAnonymousVolumes bool
	// InterruptsRunning is true when the change stops a running container.
	InterruptsRunning bool
}

// ResourceChange is a network's or volume's classified change in a ComposePlan.
type ResourceChange struct {
	Name string
	// Kind is "network" or "volume".
	Kind   string
	Action ResourceAction
}

// ComposePlan is the previewed result of `compose up` (ADR-0016, §7.12.2): the
// per-service and per-resource changes, so the operator confirms the plan rather
// than a black-box `up`. Degraded is set when convergence could not be
// determined with confidence (a missing config hash, or source-unavailable);
// Destructive is set when applying would interrupt a running container, drop an
// anonymous volume, or remove a resource — routing through the §7.4
// acknowledgement path.
type ComposePlan struct {
	Project     string
	HostRef     string
	Services    []ServiceChange
	Networks    []ResourceChange
	Volumes     []ResourceChange
	Degraded    bool
	Destructive bool
	// Notes carries plan-level labels (e.g. the source-unavailable explanation).
	Notes []string
}
