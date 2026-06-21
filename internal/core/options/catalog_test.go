package options_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/options"
)

func catalog(t *testing.T) *options.Catalog {
	t.Helper()
	c, err := options.DefaultCatalog()
	require.NoError(t, err)
	return c
}

func TestCatalogLoadsAndFlagsSecrets(t *testing.T) {
	c := catalog(t)
	assert.Equal(t, "1.51", c.APIVersion)

	// Secret-bearing options.
	for _, name := range []string{"env", "env-file", "registry-password", "build-secret"} {
		assert.True(t, c.IsSecret(name), "%q should be secret", name)
	}
	// Plainly non-secret options.
	for _, name := range []string{"user", "workdir", "tty", "publish", "restart"} {
		assert.False(t, c.IsSecret(name), "%q should not be secret", name)
	}
	// Unknown options are not secret (and not present).
	assert.False(t, c.IsSecret("nope"))
	_, ok := c.Option("nope")
	assert.False(t, ok)
}

func TestRedactReplacesOnlySecretValues(t *testing.T) {
	c := catalog(t)
	in := map[string]any{
		"env":      []string{"DB_PASSWORD=hunter2"},
		"user":     "root",
		"unknown":  "kept",
		"tty":      true,
		"env-file": "/etc/app.env",
	}
	out := c.Redact(in)

	assert.Equal(t, options.RedactedValue, out["env"])
	assert.Equal(t, options.RedactedValue, out["env-file"])
	assert.Equal(t, "root", out["user"], "non-secret untouched")
	assert.Equal(t, "kept", out["unknown"], "unknown key passes through")
	assert.Equal(t, true, out["tty"])

	// The original is not mutated and the secret string is gone from the copy.
	assert.Equal(t, []string{"DB_PASSWORD=hunter2"}, in["env"])
	for _, v := range out {
		assert.NotContains(t, toString(v), "hunter2")
	}
}

func TestRedactNilIsNil(t *testing.T) {
	assert.Nil(t, catalog(t).Redact(nil))
}

func TestValidateAcceptsAGoodSelection(t *testing.T) {
	c := catalog(t)
	require.NoError(t, c.Validate(map[string]any{
		"user":    "1000",
		"tty":     true,
		"env":     []string{"A=1"},
		"publish": []any{"127.0.0.1:8080:80"}, // []any of strings is a valid string-list
	}))
}

func TestValidateRejectsUnknownAndWrongType(t *testing.T) {
	c := catalog(t)
	assert.ErrorIs(t, c.Validate(map[string]any{"bogus": 1}), options.ErrInvalidSelection)
	assert.ErrorIs(t, c.Validate(map[string]any{"tty": "yes"}), options.ErrInvalidSelection)
	assert.ErrorIs(t, c.Validate(map[string]any{"env": "A=1"}), options.ErrInvalidSelection,
		"env is a string-list, not a string")
}

func TestValidateEnforcesRequiresAndConflicts(t *testing.T) {
	c := catalog(t)

	// registry-password requires registry-user.
	assert.ErrorIs(t, c.Validate(map[string]any{"registry-password": "s3cr3t"}), options.ErrInvalidSelection)
	require.NoError(t, c.Validate(map[string]any{"registry-password": "s3cr3t", "registry-user": "ci"}))

	// network-host conflicts with publish.
	assert.ErrorIs(t, c.Validate(map[string]any{
		"network-host": true, "publish": []string{"8080:80"},
	}), options.ErrInvalidSelection)
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []string:
		out := ""
		for _, s := range t {
			out += s
		}
		return out
	default:
		return ""
	}
}
