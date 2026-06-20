// Package compose groups a host's containers into Compose stacks using the
// project/service labels Compose stamps on them (PROJECT-BOOK §7.11.6). It is
// pure — no engine, no I/O — so the grouping that the Compose view depends on is
// unit-tested against fixtures (PROJECT-BOOK §2.5, the P7 gate).
package compose

import (
	"sort"

	"github.com/drydock/drydock/internal/core/domain"
)

// unknownService labels containers that carry a project label but no service
// label, so they are still grouped under their stack rather than dropped.
const unknownService = "(unknown)"

// Group partitions containers into Compose stacks by project label. A container
// without a project label is standalone and belongs to no stack, so it is
// omitted. Output is fully deterministic: stacks by project name, services by
// service name, containers within a service by name.
func Group(containers []domain.Container) []domain.Stack {
	type projectAcc struct {
		hostRef  string
		services map[string][]domain.Container
	}
	byProject := map[string]*projectAcc{}

	for _, c := range containers {
		if c.ComposeProject == "" {
			continue
		}
		acc := byProject[c.ComposeProject]
		if acc == nil {
			acc = &projectAcc{hostRef: c.HostRef, services: map[string][]domain.Container{}}
			byProject[c.ComposeProject] = acc
		}
		service := c.ComposeService
		if service == "" {
			service = unknownService
		}
		acc.services[service] = append(acc.services[service], c)
	}

	stacks := make([]domain.Stack, 0, len(byProject))
	for project, acc := range byProject {
		stack := domain.Stack{Project: project, HostRef: acc.hostRef}
		for name, members := range acc.services {
			sort.Slice(members, func(i, j int) bool { return members[i].Name < members[j].Name })
			svc := domain.StackService{Name: name, Containers: members, Total: len(members)}
			for _, c := range members {
				if isRunning(c) {
					svc.Running++
				}
			}
			stack.Services = append(stack.Services, svc)
			stack.Total += svc.Total
			stack.Running += svc.Running
		}
		sort.Slice(stack.Services, func(i, j int) bool { return stack.Services[i].Name < stack.Services[j].Name })
		stack.State = stateOf(stack.Running, stack.Total)
		stacks = append(stacks, stack)
	}
	sort.Slice(stacks, func(i, j int) bool { return stacks[i].Project < stacks[j].Project })
	return stacks
}

func isRunning(c domain.Container) bool { return c.State == "running" }

// stateOf reduces a stack's running/total counts to its aggregate state.
func stateOf(running, total int) domain.StackState {
	if running == 0 {
		return domain.StackStopped
	}
	if running == total {
		return domain.StackRunning
	}
	return domain.StackPartial
}
