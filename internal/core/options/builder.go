package options

import (
	"errors"
	"fmt"
)

// ErrInvalidSelection is returned by Validate when a selection references an
// unknown option, a wrong value type, a missing requirement, or a conflict.
var ErrInvalidSelection = errors.New("invalid option selection")

// Validate checks a selection against the catalog (ADR-0011, §7.5): every option
// exists, its value matches the declared type, every `requires` is satisfied,
// and no `conflicts_with` pair is both present. It validates only — assembling
// the API parameters is a separate step and never goes through a shell.
func (c *Catalog) Validate(selection map[string]any) error {
	for name, value := range selection {
		opt, ok := c.byName[name]
		if !ok {
			return fmt.Errorf("%w: unknown option %q", ErrInvalidSelection, name)
		}
		if !matchesType(opt.Type, value) {
			return fmt.Errorf("%w: option %q expects %s", ErrInvalidSelection, name, opt.Type)
		}
	}
	// Cross-option constraints, checked after every option is known to be valid.
	for name := range selection {
		opt := c.byName[name]
		for _, req := range opt.Requires {
			if _, ok := selection[req]; !ok {
				return fmt.Errorf("%w: option %q requires %q", ErrInvalidSelection, name, req)
			}
		}
		for _, conflict := range opt.ConflictsWith {
			if _, ok := selection[conflict]; ok {
				return fmt.Errorf("%w: option %q conflicts with %q", ErrInvalidSelection, name, conflict)
			}
		}
	}
	return nil
}

// matchesType reports whether value is assignable to the option's declared type.
func matchesType(t OptionType, value any) bool {
	switch t {
	case TypeString, TypePath:
		_, ok := value.(string)
		return ok
	case TypeBool:
		_, ok := value.(bool)
		return ok
	case TypeInt:
		switch value.(type) {
		case int, int64, float64:
			return true
		default:
			return false
		}
	case TypeStringList:
		return isStringList(value)
	default:
		return false
	}
}

func isStringList(value any) bool {
	switch list := value.(type) {
	case []string:
		return true
	case []any:
		for _, v := range list {
			if _, ok := v.(string); !ok {
				return false
			}
		}
		return true
	default:
		return false
	}
}
