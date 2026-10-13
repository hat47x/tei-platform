// Package beta is another independently identified, reference-only plugin.
// Its source package can be linked in the same executable as alpha.
package beta

const (
	PluginID     = "tei.reference.bundle.beta"
	CapabilityID = "process.bundle.beta"
	Version      = "1.0.0"
)

// Accept simulates receipt of an intent; it never performs the business effect.
func Accept(commandID string) (string, string) {
	return "beta accepted intent", "reference-beta://" + commandID
}
