// Package alpha is one independently identified, reference-only plugin implementation.
// It depends on neither the TEI runtime nor the other bundled plugin.
package alpha

const (
	PluginID     = "tei.reference.bundle.alpha"
	CapabilityID = "process.bundle.alpha"
	Version      = "1.0.0"
)

// Accept simulates receipt of an intent; it never performs the business effect.
func Accept(commandID string) (string, string) {
	return "alpha accepted intent", "reference-alpha://" + commandID
}
