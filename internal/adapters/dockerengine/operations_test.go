package dockerengine

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"

	"github.com/drydock/drydock/internal/core/engine"
)

func TestBuildExecOptionsUsesArgvNeverShell(t *testing.T) {
	spec := engine.ExecSpec{
		Cmd:        []string{"ls", "-la", "/var/run"},
		User:       "appuser",
		WorkingDir: "/srv",
		Tty:        true,
	}

	opts := buildExecOptions(spec)

	// The command must pass through as argv, unchanged — no shell wrapper, no
	// concatenation (ADR-0004).
	assert.Equal(t, []string{"ls", "-la", "/var/run"}, opts.Cmd)
	assert.NotContains(t, opts.Cmd, "sh")
	assert.NotContains(t, opts.Cmd, "-c")
	assert.Equal(t, "appuser", opts.User)
	assert.Equal(t, "/srv", opts.WorkingDir)
	assert.True(t, opts.Tty)
	assert.True(t, opts.AttachStdin)
	assert.True(t, opts.AttachStdout)
	assert.True(t, opts.AttachStderr)
}

func TestToSampleComputesCPUMemAndIO(t *testing.T) {
	read := time.Unix(1_700_000_000, 0).UTC()
	raw := container.StatsResponse{
		Read: read,
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 200},
			SystemUsage: 2000,
			OnlineCPUs:  2,
		},
		PreCPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 100},
			SystemUsage: 1000,
		},
		MemoryStats: container.MemoryStats{Usage: 1048576},
		Networks: map[string]container.NetworkStats{
			"eth0": {RxBytes: 500, TxBytes: 300},
		},
		BlkioStats: container.BlkioStats{
			IoServiceBytesRecursive: []container.BlkioStatEntry{
				{Op: "Read", Value: 111},
				{Op: "Write", Value: 222},
			},
		},
	}

	s := toSample("host-1", "c1", raw)

	assert.Equal(t, "host-1", s.HostRef)
	assert.Equal(t, "c1", s.ContainerID)
	assert.Equal(t, read, s.At)
	// (100/1000) * 2 cpus * 100 = 20%.
	assert.InDelta(t, 20.0, s.CPUPct, 0.001)
	assert.Equal(t, int64(1048576), s.MemBytes)
	assert.Equal(t, int64(500), s.NetRx)
	assert.Equal(t, int64(300), s.NetTx)
	assert.Equal(t, int64(111), s.BlkRead)
	assert.Equal(t, int64(222), s.BlkWrite)
}

func TestToSampleZeroDeltaYieldsZeroCPU(t *testing.T) {
	s := toSample("h", "c", container.StatsResponse{})
	assert.Equal(t, 0.0, s.CPUPct)
	assert.False(t, s.At.IsZero(), "missing read time falls back to now")
}
