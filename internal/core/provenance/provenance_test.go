package provenance_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/provenance"
)

func TestAssessLocalSignalsNoNetwork(t *testing.T) {
	p := provenance.Assess(domain.Image{
		HostRef: "h", ID: "sha256:img", Repo: "nginx", Tag: "latest",
		RepoDigest: "sha256:running", Created: time.Unix(1000, 0),
	})
	assert.Equal(t, "nginx:latest", p.ImageRef)
	assert.Equal(t, "sha256:running", p.RunningDigest)
	assert.True(t, p.Latest, ":latest is flagged as ambiguous")
	assert.False(t, p.Untagged)
	assert.False(t, p.Checked, "no registry check has happened yet")
	assert.False(t, p.TagDrifted)
}

func TestAssessUntaggedImage(t *testing.T) {
	p := provenance.Assess(domain.Image{Repo: "<none>", Tag: "<none>", Dangling: true})
	assert.True(t, p.Untagged)
	assert.False(t, p.Latest)
}

func TestWithRegistryDigestComputesDrift(t *testing.T) {
	base := provenance.Assess(domain.Image{Repo: "app", Tag: "1.2", RepoDigest: "sha256:old"})

	drifted := provenance.WithRegistryDigest(base, "sha256:new")
	assert.True(t, drifted.Checked)
	assert.True(t, drifted.TagDrifted, "running digest differs from the registry's current digest")
	assert.Equal(t, "sha256:new", drifted.RegistryDigest)

	same := provenance.WithRegistryDigest(base, "sha256:old")
	assert.True(t, same.Checked)
	assert.False(t, same.TagDrifted, "matching digests are not drifted")
}

func TestWithRegistryDigestUnknownIsNotDrift(t *testing.T) {
	// An unresolvable registry digest must not produce a false "drifted".
	base := provenance.Assess(domain.Image{Repo: "app", Tag: "1.2", RepoDigest: "sha256:old"})
	p := provenance.WithRegistryDigest(base, "")
	assert.True(t, p.Checked)
	assert.False(t, p.TagDrifted)
}
