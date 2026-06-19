package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const knownSecret = "hunter2-correct-horse-battery-staple"

func TestSensitiveAttrNeverAppearsInOutput(t *testing.T) {
	formats := []string{"json", "text"}
	keys := []string{"passphrase", "password", "token", "private_key", "data_key", "TLS_KEY"}

	for _, format := range formats {
		for _, key := range keys {
			t.Run(format+"/"+key, func(t *testing.T) {
				var buf bytes.Buffer
				log := New(&buf, Options{Level: "debug", Format: format})

				log.Info("connecting", slog.String(key, knownSecret))

				assert.NotContains(t, buf.String(), knownSecret,
					"secret leaked through key %q", key)
				assert.Contains(t, buf.String(), Redacted)
			})
		}
	}
}

func TestSensitiveAttrRedactedInsideGroup(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, Options{Level: "debug", Format: "json"})

	log.WithGroup("auth").Info("dial", slog.String("passphrase", knownSecret))

	assert.NotContains(t, buf.String(), knownSecret)
}

func TestSecretAttrRedactsRegardlessOfKey(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, Options{Level: "debug", Format: "json"})

	log.Info("dial", Secret("innocuous_name", knownSecret))

	assert.NotContains(t, buf.String(), knownSecret)
}

func TestNonSensitiveValuesPassThrough(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, Options{Level: "info", Format: "text"})

	log.Info("connected", slog.String("host", "example-host"))

	assert.Contains(t, buf.String(), "example-host")
}

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, Options{Level: "warn", Format: "text"})

	log.Debug("noisy")
	log.Info("routine")
	require.Empty(t, strings.TrimSpace(buf.String()), "below-threshold logs must be dropped")

	log.Warn("attention")
	assert.Contains(t, buf.String(), "attention")
}

func TestRedactHelper(t *testing.T) {
	assert.Equal(t, Redacted, Redact(knownSecret))
}
