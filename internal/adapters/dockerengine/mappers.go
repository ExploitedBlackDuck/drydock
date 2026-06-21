package dockerengine

import (
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"

	"github.com/drydock/drydock/internal/core/domain"
)

// Labels Compose stamps on a project's containers, used to group them into
// stacks (PROJECT-BOOK §7.11.6) and to select a project's objects for up/down.
const (
	composeProjectLabel     = "com.docker.compose.project"
	composeServiceLabel     = "com.docker.compose.service"
	composeConfigHashLabel  = "com.docker.compose.config-hash"
	composeConfigFilesLabel = "com.docker.compose.project.config_files"
	composeWorkingDirLabel  = "com.docker.compose.project.working_dir"
)

// noneRef is how the engine reports an untagged ("dangling") image.
const noneRef = "<none>:<none>"

// mapContainer converts a Docker SDK container summary into the domain type.
// hostRef identifies the host it was read from. It is pure (no I/O) so it can be
// table-tested against captured fixtures (PROJECT-BOOK §2.5).
func mapContainer(hostRef string, c container.Summary) domain.Container {
	name := ""
	if len(c.Names) > 0 {
		name = strings.TrimPrefix(c.Names[0], "/")
	}
	ports := make([]domain.Port, 0, len(c.Ports))
	for _, p := range c.Ports {
		ports = append(ports, domain.Port{
			IP:          p.IP,
			PrivatePort: p.PrivatePort,
			PublicPort:  p.PublicPort,
			Protocol:    p.Type,
		})
	}
	return domain.Container{
		ID:                 c.ID,
		HostRef:            hostRef,
		Name:               name,
		Image:              c.Image,
		State:              c.State,
		Status:             c.Status,
		Ports:              ports,
		NetworkMode:        c.HostConfig.NetworkMode,
		ComposeProject:     c.Labels[composeProjectLabel],
		ComposeService:     c.Labels[composeServiceLabel],
		ComposeConfigHash:  c.Labels[composeConfigHashLabel],
		ComposeConfigFiles: c.Labels[composeConfigFilesLabel],
		ComposeWorkingDir:  c.Labels[composeWorkingDirLabel],
		Created:            time.Unix(c.Created, 0).UTC(),
	}
}

func mapImage(hostRef string, img image.Summary) domain.Image {
	repo, tag, dangling := splitRepoTag(img.RepoTags)
	repoDigest := ""
	if len(img.RepoDigests) > 0 {
		if _, digest, found := strings.Cut(img.RepoDigests[0], "@"); found {
			repoDigest = digest
		}
	}
	return domain.Image{
		ID:         img.ID,
		HostRef:    hostRef,
		Repo:       repo,
		Tag:        tag,
		RepoDigest: repoDigest,
		Size:       img.Size,
		Dangling:   dangling,
		// Containers is -1 when the engine did not compute it; >0 means in use.
		InUse:   img.Containers > 0,
		Created: time.Unix(img.Created, 0).UTC(),
	}
}

func mapVolume(hostRef string, v *volume.Volume) domain.Volume {
	size := int64(-1)
	inUse := false
	if v.UsageData != nil {
		size = v.UsageData.Size
		inUse = v.UsageData.RefCount > 0
	}
	return domain.Volume{
		Name:       v.Name,
		HostRef:    hostRef,
		Driver:     v.Driver,
		Mountpoint: v.Mountpoint,
		Size:       size,
		InUse:      inUse,
	}
}

func mapNetwork(hostRef string, n network.Summary) domain.Network {
	return domain.Network{
		ID:      n.ID,
		HostRef: hostRef,
		Name:    n.Name,
		Driver:  n.Driver,
		// Containers is only populated on inspect; on a list it is empty, so
		// in-use is best-effort here and refined by the detail view.
		InUse: len(n.Containers) > 0,
	}
}

// splitRepoTag resolves an image's repository, tag, and dangling state from its
// RepoTags. A registry host with a port (e.g. "host:5000/img:tag") is handled by
// splitting on the final colon only when the trailing segment is a bare tag.
func splitRepoTag(repoTags []string) (repo, tag string, dangling bool) {
	ref := ""
	for _, rt := range repoTags {
		if rt != "" && rt != noneRef {
			ref = rt
			break
		}
	}
	if ref == "" {
		return "<none>", "<none>", true
	}

	idx := strings.LastIndex(ref, ":")
	// A colon that belongs to a registry port (segment after it contains "/")
	// is not a tag separator.
	if idx < 0 || strings.Contains(ref[idx+1:], "/") {
		return ref, "latest", false
	}
	return ref[:idx], ref[idx+1:], false
}
