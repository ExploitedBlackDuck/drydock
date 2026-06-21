package options

import (
	"embed"
	"fmt"

	"github.com/BurntSushi/toml"
)

//go:embed catalogs/*.toml
var catalogFiles embed.FS

// RedactedValue replaces a secret-flagged option's value when it is captured
// into the operations record, the audit log, or logs (ADR-0023). It is recorded
// as present-but-redacted so history shows that a value was set without leaking
// it.
const RedactedValue = "‹redacted›"

// OptionType is an option's value type, used by the builder to validate a
// selection before it is assembled into API parameters.
type OptionType string

// The catalogued option value types the builder validates against.
const (
	TypeString     OptionType = "string"
	TypeBool       OptionType = "bool"
	TypeInt        OptionType = "int"
	TypeStringList OptionType = "string-list"
	TypePath       OptionType = "path"
)

// Risk classifies an option by the worst outcome selecting it can cause.
type Risk string

// The risk classes an option can carry, worst-outcome first in meaning.
const (
	RiskRead        Risk = "read"
	RiskMutating    Risk = "mutating"
	RiskDestructive Risk = "destructive"
)

// Option is one catalogued operation option (PROJECT-BOOK §7.5).
type Option struct {
	Name          string     `toml:"name"`
	Type          OptionType `toml:"type"`
	Default       any        `toml:"default"`
	Category      string     `toml:"category"`
	Summary       string     `toml:"summary"`
	Description   string     `toml:"description"`
	Risk          Risk       `toml:"risk"`
	AffectsData   bool       `toml:"affects_data"`
	Secret        bool       `toml:"secret"`
	ConflictsWith []string   `toml:"conflicts_with"`
	Requires      []string   `toml:"requires"`
	Impacts       []string   `toml:"impacts"`
}

type catalogFile struct {
	Schema     int      `toml:"schema"`
	APIVersion string   `toml:"api_version"`
	Options    []Option `toml:"option"`
}

// Catalog is the typed, validated set of options for one engine API version.
type Catalog struct {
	SchemaVersion int
	APIVersion    string
	byName        map[string]Option
}

// LoadCatalog parses the embedded catalog file (e.g. "docker@1.51.toml").
func LoadCatalog(name string) (*Catalog, error) {
	data, err := catalogFiles.ReadFile("catalogs/" + name)
	if err != nil {
		return nil, fmt.Errorf("reading catalog %q: %w", name, err)
	}
	var cf catalogFile
	if err := toml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("parsing catalog %q: %w", name, err)
	}
	byName := make(map[string]Option, len(cf.Options))
	for _, opt := range cf.Options {
		if opt.Name == "" {
			return nil, fmt.Errorf("catalog %q has an option with no name", name)
		}
		if _, dup := byName[opt.Name]; dup {
			return nil, fmt.Errorf("catalog %q has duplicate option %q", name, opt.Name)
		}
		byName[opt.Name] = opt
	}
	return &Catalog{SchemaVersion: cf.Schema, APIVersion: cf.APIVersion, byName: byName}, nil
}

// DefaultCatalog loads the catalog Drydock ships with. The asset is embedded, so
// a parse failure is a build defect.
func DefaultCatalog() (*Catalog, error) {
	return LoadCatalog("docker@1.51.toml")
}

// Option returns the named option and whether it exists.
func (c *Catalog) Option(name string) (Option, bool) {
	opt, ok := c.byName[name]
	return opt, ok
}

// IsSecret reports whether the named option carries secret material.
func (c *Catalog) IsSecret(name string) bool {
	opt, ok := c.byName[name]
	return ok && opt.Secret
}

// Redact returns a copy of an option set with every secret-flagged value
// replaced by RedactedValue (ADR-0023). Unknown and non-secret keys pass
// through unchanged. It is the capture-path boundary: callers redact before
// persisting an operation, writing an audit entry, or logging.
func (c *Catalog) Redact(set map[string]any) map[string]any {
	if set == nil {
		return nil
	}
	out := make(map[string]any, len(set))
	for key, value := range set {
		if c.IsSecret(key) {
			out[key] = RedactedValue
		} else {
			out[key] = value
		}
	}
	return out
}
