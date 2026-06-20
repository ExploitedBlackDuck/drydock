package options_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/options"
)

func hasCode(a options.Assessment, code string) bool {
	for _, d := range a.Decisions {
		if d.Code == code {
			return true
		}
	}
	return false
}

func TestObserveModeBlocksMutation(t *testing.T) {
	a := options.Assess(options.Request{Kind: domain.OpContainerStop, ObserveMode: true})
	assert.True(t, a.Blocked)
	assert.True(t, hasCode(a, "ERR_OBSERVE_MODE"))
}

func TestForceRemoveRunningRequiresAck(t *testing.T) {
	a := options.Assess(options.Request{
		Kind: domain.OpContainerRemove, Force: true, TargetRunning: true,
	})
	assert.True(t, a.RequiresAck)
	assert.False(t, a.Blocked)
	assert.True(t, hasCode(a, "FORCE_REMOVE_RUNNING"))
}

func TestRemoveStoppedContainerNoAck(t *testing.T) {
	a := options.Assess(options.Request{Kind: domain.OpContainerRemove, TargetRunning: false})
	assert.False(t, a.RequiresAck, "removing a stopped container does not trip rm -f rule")
}

func TestComposeDownVolumesRequiresAck(t *testing.T) {
	a := options.Assess(options.Request{Kind: domain.OpComposeDown, Volumes: true})
	assert.True(t, a.RequiresAck)
	assert.True(t, hasCode(a, "COMPOSE_DOWN_VOLUMES"))
}

func TestExecAsRootWarnsOnly(t *testing.T) {
	a := options.Assess(options.Request{Kind: domain.OpContainerExec, RunAsRoot: true})
	assert.False(t, a.Blocked)
	assert.False(t, a.RequiresAck)
	assert.True(t, hasCode(a, "EXEC_ROOT"))
}

func TestPruneIsDestructive(t *testing.T) {
	a := options.Assess(options.Request{Kind: domain.OpImagePrune})
	assert.True(t, a.RequiresAck)
}

func TestReadLikeOperationHasNoDecisions(t *testing.T) {
	a := options.Assess(options.Request{Kind: domain.OpContainerStart})
	assert.False(t, a.Blocked)
	assert.False(t, a.RequiresAck)
	assert.Empty(t, a.Decisions)
}
