// Package options implements the declarative impact-rule engine (ADR-0011,
// PROJECT-BOOK §7.5): given an operation and its flags, it produces warnings,
// acknowledgement requirements, or an outright block — pure, table-tested rules
// the UI and the core both consult before a destructive action runs.
package options

import "github.com/drydock/drydock/internal/core/domain"

// Level is the severity of an impact decision.
type Level string

const (
	// LevelWarn surfaces a caution but does not gate the operation.
	LevelWarn Level = "warn"
	// LevelRequireAck requires an explicit acknowledgement before proceeding.
	LevelRequireAck Level = "require_ack"
	// LevelBlock forbids the operation entirely.
	LevelBlock Level = "block"
)

// Decision is one rule outcome.
type Decision struct {
	Level   Level
	Code    string
	Message string
}

// Request describes the operation being assessed.
type Request struct {
	Kind          domain.OperationKind
	ObserveMode   bool
	Force         bool // e.g. rm -f
	TargetRunning bool // the container is running
	Volumes       bool // prune/down includes volumes
	RunAsRoot     bool // exec as root / privileged
}

// Assessment is the aggregate of all triggered decisions.
type Assessment struct {
	Decisions   []Decision
	Blocked     bool
	RequiresAck bool
}

// rule is a pure predicate that may contribute a decision.
type rule func(Request) (Decision, bool)

// rules is the ordered rule set. Each is small and independently testable.
var rules = []rule{
	// Observe-mode hosts reject every mutation (ADR-0013).
	func(r Request) (Decision, bool) {
		if r.ObserveMode && isMutation(r.Kind) {
			return Decision{LevelBlock, "ERR_OBSERVE_MODE", "Host is observe-only; mutations are blocked."}, true
		}
		return Decision{}, false
	},
	// Prune/remove that includes volumes: named volumes hold persistent data and
	// are confirmed individually (routes to the per-volume flow, §7.4).
	func(r Request) (Decision, bool) {
		if r.Volumes && (r.Kind == domain.OpComposeDown || r.Kind == domain.OpImagePrune || r.Kind == domain.OpContainerPrune) {
			return Decision{LevelRequireAck, "VOLUMES_INCLUDED", "Named volumes hold persistent data; each will be confirmed individually."}, true
		}
		return Decision{}, false
	},
	// `rm -f` on a running container loses in-flight work.
	func(r Request) (Decision, bool) {
		if r.Kind == domain.OpContainerRemove && r.Force && r.TargetRunning {
			return Decision{LevelRequireAck, "FORCE_REMOVE_RUNNING", "Force-removes a running container; in-flight work is lost."}, true
		}
		return Decision{}, false
	},
	// `compose down -v` deletes the stack's volumes.
	func(r Request) (Decision, bool) {
		if r.Kind == domain.OpComposeDown && r.Volumes {
			return Decision{LevelRequireAck, "COMPOSE_DOWN_VOLUMES", "Removes the stack's volumes; persistent data is deleted."}, true
		}
		return Decision{}, false
	},
	// Other destructive operations require acknowledgement.
	func(r Request) (Decision, bool) {
		if r.Kind.Destructive() && r.Kind != domain.OpContainerRemove {
			return Decision{LevelRequireAck, "DESTRUCTIVE", "This operation removes data and cannot be undone."}, true
		}
		return Decision{}, false
	},
	// Exec as root / privileged is a caution, not a gate.
	func(r Request) (Decision, bool) {
		if r.Kind == domain.OpContainerExec && r.RunAsRoot {
			return Decision{LevelWarn, "EXEC_ROOT", "Runs as root inside the container."}, true
		}
		return Decision{}, false
	},
}

// Assess runs every rule and aggregates the decisions.
func Assess(req Request) Assessment {
	var a Assessment
	for _, rule := range rules {
		decision, ok := rule(req)
		if !ok {
			continue
		}
		a.Decisions = append(a.Decisions, decision)
		switch decision.Level {
		case LevelBlock:
			a.Blocked = true
		case LevelRequireAck:
			a.RequiresAck = true
		case LevelWarn:
		}
	}
	return a
}

func isMutation(kind domain.OperationKind) bool {
	switch kind {
	case domain.OpComposeUp:
		return true
	default:
		// Every other catalogued kind is a mutation; reads do not pass through
		// the rule engine.
		return kind != ""
	}
}
