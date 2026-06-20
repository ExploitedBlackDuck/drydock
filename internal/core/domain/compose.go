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
