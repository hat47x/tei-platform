package formatversion

import (
	"fmt"
	"sort"
	"strings"
)

// Formats is the plugin codec registration entry (#1920). A plugin registers
// its format identifier and one codec per version; the core owns dispatch,
// classification and the invariants.
//
// Format identifiers are stable strings chosen by the owner of the format (for
// example tei.semantic). Two formats may use the same version numbers without
// colliding, and registering the same format/version twice is refused.
type Formats struct {
	formats map[string]*Registry
}

// NewFormats returns an empty format registry.
func NewFormats() *Formats {
	return &Formats{formats: map[string]*Registry{}}
}

// Register adds a codec for a format. The first registration for a format
// creates its per-version registry.
func (f *Formats) Register(format string, codec Codec) error {
	format = strings.TrimSpace(format)
	if format == "" {
		return fmt.Errorf("formatversion: empty format identifier")
	}
	registry, ok := f.formats[format]
	if !ok {
		registry = NewRegistry()
		f.formats[format] = registry
	}
	return registry.Register(codec)
}

// Registry returns the per-version registry for a format. It is nil when the
// format is not registered.
func (f *Formats) Registry(format string) *Registry {
	return f.formats[format]
}

// Supported lists the registered format identifiers in ascending order.
func (f *Formats) Supported() []string {
	out := make([]string, 0, len(f.formats))
	for format := range f.formats {
		out = append(out, format)
	}
	sort.Strings(out)
	return out
}

// Convert dispatches to the format's registry. An unknown format fails as a
// unit of work and names the registered formats.
func (f *Formats) Convert(format string, source []byte, from, to Version) (Result, error) {
	registry, ok := f.formats[format]
	if !ok {
		result := Result{From: from, To: to, Source: append([]byte(nil), source...)}
		return systemFailure(result, "UNKNOWN_FORMAT",
			fmt.Sprintf("no codec registered for format %q; registered formats: %s", format, f.supportedText()))
	}
	return registry.Convert(source, from, to)
}

func (f *Formats) supportedText() string {
	formats := f.Supported()
	if len(formats) == 0 {
		return "none"
	}
	return strings.Join(formats, ", ")
}
