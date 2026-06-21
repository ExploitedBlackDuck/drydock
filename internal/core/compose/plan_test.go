package compose_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/compose"
	"github.com/drydock/drydock/internal/core/domain"
)

func serviceChange(plan domain.ComposePlan, name string) (domain.ServiceChange, bool) {
	for _, s := range plan.Services {
		if s.Service == name {
			return s, true
		}
	}
	return domain.ServiceChange{}, false
}

func TestPlanClassifiesEachServiceAction(t *testing.T) {
	desired := &compose.DesiredStack{
		Services: []compose.DesiredService{
			{Name: "new", ConfigHash: "h-new"},
			{Name: "stopped", ConfigHash: "h1"},
			{Name: "running", ConfigHash: "h2"},
			{Name: "changed", ConfigHash: "h3-new"},
		},
	}
	observed := compose.ObservedStack{
		Services: []compose.ObservedService{
			{Name: "stopped", Running: false, ConfigHash: "h1"},
			{Name: "running", Running: true, ConfigHash: "h2"},
			{Name: "changed", Running: true, ConfigHash: "h3-old"},
		},
	}

	plan := compose.Plan("app", "local", desired, observed)
	require.False(t, plan.Degraded)

	create, _ := serviceChange(plan, "new")
	assert.Equal(t, domain.ServiceCreate, create.Action)
	start, _ := serviceChange(plan, "stopped")
	assert.Equal(t, domain.ServiceStart, start.Action)
	noop, _ := serviceChange(plan, "running")
	assert.Equal(t, domain.ServiceNoop, noop.Action)
	recreate, _ := serviceChange(plan, "changed")
	assert.Equal(t, domain.ServiceRecreate, recreate.Action)
	assert.True(t, recreate.InterruptsRunning, "recreating a running service interrupts it")
}

func TestPlanRecreateDroppingAnonymousVolumeIsDestructive(t *testing.T) {
	desired := &compose.DesiredStack{
		Services: []compose.DesiredService{
			{Name: "db", ConfigHash: "new", HasAnonymousVolumes: true},
		},
	}
	observed := compose.ObservedStack{
		Services: []compose.ObservedService{{Name: "db", Running: true, ConfigHash: "old"}},
	}

	plan := compose.Plan("app", "local", desired, observed)
	change, ok := serviceChange(plan, "db")
	require.True(t, ok)
	assert.Equal(t, domain.ServiceRecreate, change.Action)
	assert.True(t, change.DropsAnonymousVolumes)
	assert.True(t, plan.Destructive, "dropping an anonymous volume makes the plan destructive (require_ack)")
}

func TestPlanDegradesWhenConfigHashUnavailable(t *testing.T) {
	// No config-hash label on the observed container: fall back to a coarse,
	// clearly-labelled image diff rather than asserting false precision.
	desired := &compose.DesiredStack{
		Services: []compose.DesiredService{{Name: "web", Image: "nginx:1.27"}},
	}
	observed := compose.ObservedStack{
		Services: []compose.ObservedService{{Name: "web", Running: true, ConfigHash: "", Image: "nginx:1.25"}},
	}

	plan := compose.Plan("app", "local", desired, observed)
	assert.True(t, plan.Degraded, "a missing config hash forces a degraded plan")
	change, _ := serviceChange(plan, "web")
	assert.Equal(t, domain.ServiceRecreate, change.Action, "coarse diff sees the image change")
	assert.Contains(t, change.Reasons[0], "coarse")
}

func TestPlanSourceUnavailableIsLabelledNotFalseNoChange(t *testing.T) {
	observed := compose.ObservedStack{
		Services: []compose.ObservedService{
			{Name: "web", Running: true},
			{Name: "db", Running: true},
		},
	}

	// A nil desired stack means the project source was not accessible.
	plan := compose.Plan("app", "remote", nil, observed)
	assert.True(t, plan.Degraded)
	require.NotEmpty(t, plan.Notes)
	assert.Contains(t, plan.Notes[0], "source unavailable")

	// Every service is labelled "cannot determine" — never a clean no-op.
	require.Len(t, plan.Services, 2)
	for _, s := range plan.Services {
		require.NotEmpty(t, s.Reasons)
		assert.Contains(t, s.Reasons[0], "source unavailable")
	}
}

func TestPlanCreatesMissingNetworksAndVolumes(t *testing.T) {
	desired := &compose.DesiredStack{
		Networks: []string{"frontend", "backend"},
		Volumes:  []string{"db-data"},
	}
	observed := compose.ObservedStack{
		Networks: []string{"frontend"},
		Volumes:  []string{},
	}

	plan := compose.Plan("app", "local", desired, observed)
	require.Len(t, plan.Networks, 1)
	assert.Equal(t, "backend", plan.Networks[0].Name)
	assert.Equal(t, domain.ResourceCreate, plan.Networks[0].Action)
	require.Len(t, plan.Volumes, 1)
	assert.Equal(t, "db-data", plan.Volumes[0].Name)
	assert.False(t, plan.Destructive, "creating missing resources is not destructive")
}

func TestPlanAllUpToDateIsCleanNoop(t *testing.T) {
	desired := &compose.DesiredStack{
		Services: []compose.DesiredService{{Name: "web", ConfigHash: "h"}},
	}
	observed := compose.ObservedStack{
		Services: []compose.ObservedService{{Name: "web", Running: true, ConfigHash: "h"}},
	}
	plan := compose.Plan("app", "local", desired, observed)
	assert.False(t, plan.Degraded)
	assert.False(t, plan.Destructive)
	change, _ := serviceChange(plan, "web")
	assert.Equal(t, domain.ServiceNoop, change.Action)
}
