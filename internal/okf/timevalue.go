package okf

import (
	"regexp"
	"strings"
	"time"
)

// isoTimestampPattern accepts an ISO 8601 datetime. OKF v0.2 §5 requires every
// timestamp-valued key to carry an explicit UTC offset, so the offset group is
// checked separately rather than folded into the pattern.
var isoTimestampPattern = regexp.MustCompile(
	`^(\d{4})-(\d{2})-(\d{2})[Tt](\d{2}):(\d{2}):(\d{2})(\.\d+)?([Zz]|[+-]\d{2}:\d{2})$`)

// dateOnlyPattern matches a bare YYYY-MM-DD, which names a different instant in
// every timezone. The reader refuses to guess one.
var dateOnlyPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// parseISODateTime parses a timestamp-valued frontmatter value. It returns the
// instant, whether an explicit offset was present, and whether the value parsed
// at all. A date-only value, or a datetime without an offset, parses to ok=true
// with hasOffset=false so the caller can name the specific problem.
func parseISODateTime(raw string) (instant time.Time, hasOffset bool, ok bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false, false
	}
	if dateOnlyPattern.MatchString(value) {
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			return time.Time{}, false, false
		}
		return parsed, false, true
	}
	// A datetime without an offset still parses when a layout omits the zone; try
	// it so the caller can report "offset missing" rather than "invalid".
	if match := isoTimestampPattern.FindStringSubmatch(value); match != nil {
		parsed, err := time.Parse(time.RFC3339Nano, normalizeTimestampForParse(value))
		if err != nil {
			// A lowercase t/z is accepted by the pattern but not by RFC3339Nano's
			// strict layout, so retry after upper-casing the separators only.
			parsed, err = time.Parse(time.RFC3339Nano, strings.ToUpper(value[:1])+value[1:])
		}
		if err != nil {
			return time.Time{}, true, false
		}
		return parsed, true, true
	}
	for _, layout := range []string{"2006-01-02T15:04:05.999999999", "2006-01-02T15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, false, true
		}
	}
	return time.Time{}, false, false
}

// normalizeTimestampForParse upper-cases the T and Z separators so RFC3339Nano
// accepts a lowercase timestamp without otherwise rewriting the author's value.
func normalizeTimestampForParse(value string) string {
	out := value
	if index := strings.IndexAny(out, "Tt"); index >= 0 {
		out = out[:index] + "T" + out[index+1:]
	}
	if strings.HasSuffix(out, "z") {
		out = out[:len(out)-1] + "Z"
	}
	return out
}

// clockNow is the staleness reference instant. It is a variable so a test can pin
// a fixed time without touching the clock package.
var clockNow = func() time.Time { return time.Now().UTC() }
