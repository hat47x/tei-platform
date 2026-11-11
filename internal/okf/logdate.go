package okf

import "regexp"

// okfLogDateHeading matches exactly YYYY-MM-DD, the date heading form OKF v0.2
// section 9 requires in a log.md.
func okfLogDateHeading() *regexp.Regexp {
	return regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
}
