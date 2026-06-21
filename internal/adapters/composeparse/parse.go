// Package composeparse loads a Compose project's files into the neutral
// compose.DesiredStack the plan engine diffs against observed state (P10,
// ADR-0016). It uses the pinned compose-go parser — never a shell — and reads
// files from the local filesystem only: Drydock does not read arbitrary files
// off a remote host (ADR-0005), so the caller supplies locally-accessible paths
// (a local host, or compose files the operator points it at).
package composeparse

import (
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"

	"github.com/drydock/drydock/internal/core/compose"
)

// Parse loads the project's compose files into a DesiredStack. ConfigHash is
// deliberately left empty: Compose's per-service runtime hash is out-of-spec and
// version-fragile to reproduce (ADR-0016), so the plan falls back to a labelled
// coarse diff rather than asserting a precise hash match.
func Parse(ctx context.Context, projectName, workingDir string, files []string) (compose.DesiredStack, error) {
	configFiles := make([]types.ConfigFile, len(files))
	for i, f := range files {
		configFiles[i] = types.ConfigFile{Filename: f}
	}

	project, err := loader.LoadWithContext(ctx, types.ConfigDetails{
		WorkingDir:  workingDir,
		ConfigFiles: configFiles,
		Environment: types.Mapping{},
	}, func(o *loader.Options) {
		o.SetProjectName(projectName, true)
		o.SkipConsistencyCheck = true
	})
	if err != nil {
		return compose.DesiredStack{}, fmt.Errorf("loading compose project %q: %w", projectName, err)
	}

	stack := compose.DesiredStack{}
	for name, service := range project.Services {
		desired := compose.DesiredService{Name: name, Image: service.Image}
		for _, vol := range service.Volumes {
			// A volume mount with no source is an anonymous volume — a recreate
			// drops it (the data-loss case the plan must flag).
			if vol.Type == types.VolumeTypeVolume && vol.Source == "" {
				desired.HasAnonymousVolumes = true
			}
		}
		stack.Services = append(stack.Services, desired)
	}
	for name := range project.Networks {
		stack.Networks = append(stack.Networks, name)
	}
	for name := range project.Volumes {
		stack.Volumes = append(stack.Volumes, name)
	}
	return stack, nil
}
