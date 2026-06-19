package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "absent.toml"))
	require.NoError(t, err)
	assert.Equal(t, Default(), cfg)
}

func TestLoadValidFileOverridesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := `
[log]
level = "debug"
format = "json"

[history]
resource_retention = "6h"
`
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.Equal(t, 6*time.Hour, cfg.History.ResourceRetention.Duration)
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := map[string]string{
		"bad level":    "[log]\nlevel = \"loud\"\nformat = \"json\"\n",
		"bad format":   "[log]\nlevel = \"info\"\nformat = \"yaml\"\n",
		"bad duration": "[history]\nresource_retention = \"-1h\"\n",
		"malformed":    "this is not = valid = toml",
	}
	for name, contents := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))

			_, err := Load(path)
			assert.Error(t, err)
		})
	}
}

func TestResolvePathsAreAbsoluteAndNamespaced(t *testing.T) {
	paths, err := ResolvePaths()
	require.NoError(t, err)

	assert.True(t, filepath.IsAbs(paths.ConfigDir), "config dir must be absolute")
	assert.True(t, filepath.IsAbs(paths.DataDir), "data dir must be absolute")
	assert.Equal(t, appDir, filepath.Base(paths.ConfigDir))
	assert.Equal(t, appDir, filepath.Base(paths.DataDir))
	assert.Equal(t, "config.toml", filepath.Base(paths.ConfigFile()))
}

func TestEnsureDirsCreatesRestrictedDirectories(t *testing.T) {
	base := t.TempDir()
	paths := Paths{
		ConfigDir: filepath.Join(base, "cfg", appDir),
		DataDir:   filepath.Join(base, "data", appDir),
	}
	require.NoError(t, paths.EnsureDirs())

	for _, dir := range []string{paths.ConfigDir, paths.DataDir} {
		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
		assert.Equal(t, dirPerm, info.Mode().Perm(), "directory must be owner-only")
	}
}
