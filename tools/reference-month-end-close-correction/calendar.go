package monthendclose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// Schema identifiers of the sample's declarative inputs and outputs.
const (
	CalendarSchema = "tei.reference.mecc-calendar/v0"
	RuleSchema     = "tei.reference.mecc-close-rule/v0"
	DecisionSchema = "tei.reference.mecc-close-decision/v0"
)

// Holiday is one non-business date of a calendar version.
type Holiday struct {
	Date string `json:"date"`
	Name string `json:"name"`
}

// Calendar is one immutable version of a business-day calendar. A revision
// is a new version; an existing version never changes (its digest is
// recorded with it and checked on load).
type Calendar struct {
	Schema         string    `json:"schema"`
	ID             string    `json:"id"`
	Version        int       `json:"version"`
	Region         string    `json:"region"`
	Timezone       string    `json:"timezone"`
	WeeklyHolidays []string  `json:"weeklyHolidays"`
	Holidays       []Holiday `json:"holidays"`
	// RecordedAt is when this version became known. A decision evaluated
	// before it cannot use it.
	RecordedAt string `json:"recordedAt"`
	Revises    int    `json:"revises,omitempty"`
	Digest     string `json:"digest,omitempty"`
}

// CloseRule declares how the close date, the file cut-off, the close job and
// payment due dates are derived from a calendar.
type CloseRule struct {
	Schema string `json:"schema"`
	// CloseDay is "last-business-day" (of the target month) or
	// "nth-business-day" (N-th business day of the target month).
	CloseDay string `json:"closeDay"`
	N        int    `json:"n,omitempty"`
	// CutoffLocal is the local wall time of the file cut-off on the close date.
	CutoffLocal string `json:"cutoffLocal"`
	// CloseJobBusinessDaysAfter is the number of business days after the
	// close date on which the close job starts at CloseJobLocal.
	CloseJobBusinessDaysAfter int    `json:"closeJobBusinessDaysAfter"`
	CloseJobLocal             string `json:"closeJobLocal"`
	// DueDateAdjustment is applied to a nominal due date that is not a
	// business day. Only "previous-business-day" is implemented.
	DueDateAdjustment string `json:"dueDateAdjustment"`
	// BusinessDateRule decides the business date of an accepted order. Only
	// "local-date-of-acceptance" (the date in the calendar's timezone) is
	// implemented; the server's or the database session's zone is never used.
	BusinessDateRule string `json:"businessDateRule"`
}

// Payable is an unpaid item whose due date depends on the calendar.
type Payable struct {
	ID         string `json:"id"`
	NominalDue string `json:"nominalDue"`
}

// DueDate is the computed due date of one payable.
type DueDate struct {
	PayableID  string `json:"payableId"`
	NominalDue string `json:"nominalDue"`
	Due        string `json:"due"`
}

// ScheduledJob is one job time derived from the calendar.
type ScheduledJob struct {
	Name string `json:"name"`
	At   string `json:"at"`
}

// Decision is the output of step 1. Every later step receives it and checks
// its digest, its calendar version and that it is the active decision of the
// month.
type Decision struct {
	Schema           string         `json:"schema"`
	TargetMonth      string         `json:"targetMonth"`
	CalendarID       string         `json:"calendarId"`
	CalendarVersion  int            `json:"calendarVersion"`
	CalendarDigest   string         `json:"calendarDigest"`
	Region           string         `json:"region"`
	Timezone         string         `json:"timezone"`
	EvaluatedAt      string         `json:"evaluatedAt"`
	Rule             CloseRule      `json:"rule"`
	CloseDate        string         `json:"closeDate"`
	CutoffAt         string         `json:"cutoffAt"`
	Jobs             []ScheduledJob `json:"jobs"`
	DueDates         []DueDate      `json:"dueDates"`
	BusinessDateRule string         `json:"businessDateRule"`
	Digest           string         `json:"digest,omitempty"`
}

// Job names of a decision.
const (
	JobFileCutoff = "file-cutoff"
	JobClose      = "close-job"
)

const dateLayout = "2006-01-02"

// ErrCalendarNotYetKnown is returned when a decision would use a calendar
// version recorded after the evaluation instant.
var ErrCalendarNotYetKnown = errors.New("calendar version recorded after the evaluation instant")

// ErrDigestMismatch is returned when a stored or received document does not
// match its digest.
var ErrDigestMismatch = errors.New("digest mismatch")

func digestOf(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// CalendarDigest is the SHA-256 of the calendar without its digest field,
// with holidays sorted by date.
func CalendarDigest(c Calendar) string {
	c.Digest = ""
	c.Holidays = append([]Holiday(nil), c.Holidays...)
	sort.Slice(c.Holidays, func(i, j int) bool { return c.Holidays[i].Date < c.Holidays[j].Date })
	return digestOf(c)
}

// DecisionDigest is the SHA-256 of the decision without its digest field.
func DecisionDigest(d Decision) string {
	d.Digest = ""
	return digestOf(d)
}

// ValidateCalendar checks the declarative fields of a calendar version.
func ValidateCalendar(c Calendar) error {
	if c.Schema != CalendarSchema {
		return fmt.Errorf("calendar schema %q, want %q", c.Schema, CalendarSchema)
	}
	if c.ID == "" || c.Version < 1 || c.Region == "" {
		return fmt.Errorf("calendar id, version >= 1 and region are required")
	}
	if _, err := time.LoadLocation(c.Timezone); err != nil || c.Timezone == "" {
		return fmt.Errorf("calendar timezone %q: %v", c.Timezone, err)
	}
	if _, err := time.Parse(time.RFC3339, c.RecordedAt); err != nil {
		return fmt.Errorf("calendar recordedAt: %w", err)
	}
	for _, w := range c.WeeklyHolidays {
		if _, ok := weekdays[w]; !ok {
			return fmt.Errorf("unknown weekly holiday %q", w)
		}
	}
	seen := map[string]bool{}
	for _, h := range c.Holidays {
		if _, err := time.Parse(dateLayout, h.Date); err != nil {
			return fmt.Errorf("holiday %q: %w", h.Date, err)
		}
		if seen[h.Date] {
			return fmt.Errorf("holiday %s listed twice", h.Date)
		}
		seen[h.Date] = true
	}
	if c.Digest != "" && c.Digest != CalendarDigest(c) {
		return fmt.Errorf("calendar %s v%d: %w", c.ID, c.Version, ErrDigestMismatch)
	}
	return nil
}

var weekdays = map[string]time.Weekday{
	"Sunday": time.Sunday, "Monday": time.Monday, "Tuesday": time.Tuesday, "Wednesday": time.Wednesday,
	"Thursday": time.Thursday, "Friday": time.Friday, "Saturday": time.Saturday,
}

// IsBusinessDay reports whether the date (a civil date, no zone) is a
// business day in the calendar version.
func (c Calendar) IsBusinessDay(d time.Time) bool {
	for _, w := range c.WeeklyHolidays {
		if d.Weekday() == weekdays[w] {
			return false
		}
	}
	day := d.Format(dateLayout)
	for _, h := range c.Holidays {
		if h.Date == day {
			return false
		}
	}
	return true
}

func civil(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func parseCivil(s string) (time.Time, error) { return time.Parse(dateLayout, s) }

// Revise returns the next version of the calendar with added holidays.
func (c Calendar) Revise(add []Holiday, recordedAt string) (Calendar, error) {
	n := c
	n.Version = c.Version + 1
	n.Revises = c.Version
	n.RecordedAt = recordedAt
	n.Holidays = append(append([]Holiday(nil), c.Holidays...), add...)
	n.Digest = ""
	if err := ValidateCalendar(n); err != nil {
		return Calendar{}, err
	}
	n.Digest = CalendarDigest(n)
	return n, nil
}

func parseLocalClock(s string) (int, int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, 0, fmt.Errorf("local time %q: %w", s, err)
	}
	return t.Hour(), t.Minute(), nil
}

// ComputeDecision derives step 1's decision. It is a pure function of its
// inputs: the calendar version, the rule, the target month, the explicit
// evaluation instant and the payables.
func ComputeDecision(cal Calendar, rule CloseRule, month string, evaluatedAt time.Time, payables []Payable) (Decision, error) {
	if err := ValidateCalendar(cal); err != nil {
		return Decision{}, err
	}
	if rule.Schema != RuleSchema {
		return Decision{}, fmt.Errorf("rule schema %q, want %q", rule.Schema, RuleSchema)
	}
	if rule.BusinessDateRule != BusinessDateLocal {
		return Decision{}, fmt.Errorf("business date rule %q is not implemented", rule.BusinessDateRule)
	}
	if rule.DueDateAdjustment != "previous-business-day" {
		return Decision{}, fmt.Errorf("due date adjustment %q is not implemented", rule.DueDateAdjustment)
	}
	recorded, _ := time.Parse(time.RFC3339, cal.RecordedAt)
	if evaluatedAt.Before(recorded) {
		return Decision{}, fmt.Errorf("%w: %s v%d recorded %s, evaluated %s", ErrCalendarNotYetKnown, cal.ID, cal.Version, cal.RecordedAt, evaluatedAt.Format(time.RFC3339))
	}
	loc, _ := time.LoadLocation(cal.Timezone)
	first, err := time.Parse("2006-01", month)
	if err != nil {
		return Decision{}, fmt.Errorf("target month %q: %w", month, err)
	}
	first = civil(first.Year(), first.Month(), 1)
	var closeDate time.Time
	switch rule.CloseDay {
	case "last-business-day":
		d := first.AddDate(0, 1, -1)
		for !cal.IsBusinessDay(d) {
			d = d.AddDate(0, 0, -1)
		}
		closeDate = d
	case "nth-business-day":
		if rule.N < 1 {
			return Decision{}, fmt.Errorf("nth-business-day needs n >= 1")
		}
		d, k := first, 0
		for {
			if cal.IsBusinessDay(d) {
				k++
				if k == rule.N {
					break
				}
			}
			d = d.AddDate(0, 0, 1)
		}
		closeDate = d
	default:
		return Decision{}, fmt.Errorf("close day rule %q is not implemented", rule.CloseDay)
	}
	if closeDate.Month() != first.Month() {
		return Decision{}, fmt.Errorf("no business day for the close in %s", month)
	}
	ch, cm, err := parseLocalClock(rule.CutoffLocal)
	if err != nil {
		return Decision{}, err
	}
	cutoff := time.Date(closeDate.Year(), closeDate.Month(), closeDate.Day(), ch, cm, 0, 0, loc)
	jobDay := closeDate
	for k := 0; k < rule.CloseJobBusinessDaysAfter; {
		jobDay = jobDay.AddDate(0, 0, 1)
		if cal.IsBusinessDay(jobDay) {
			k++
		}
	}
	jh, jm, err := parseLocalClock(rule.CloseJobLocal)
	if err != nil {
		return Decision{}, err
	}
	jobAt := time.Date(jobDay.Year(), jobDay.Month(), jobDay.Day(), jh, jm, 0, 0, loc)
	d := Decision{
		Schema: DecisionSchema, TargetMonth: month, CalendarID: cal.ID, CalendarVersion: cal.Version,
		CalendarDigest: CalendarDigest(cal), Region: cal.Region, Timezone: cal.Timezone,
		EvaluatedAt: evaluatedAt.In(loc).Format(time.RFC3339), Rule: rule,
		CloseDate: closeDate.Format(dateLayout), CutoffAt: cutoff.Format(time.RFC3339),
		Jobs:             []ScheduledJob{{JobFileCutoff, cutoff.Format(time.RFC3339)}, {JobClose, jobAt.Format(time.RFC3339)}},
		BusinessDateRule: rule.BusinessDateRule,
	}
	ps := append([]Payable(nil), payables...)
	sort.Slice(ps, func(i, j int) bool { return ps[i].ID < ps[j].ID })
	for _, p := range ps {
		nd, err := parseCivil(p.NominalDue)
		if err != nil {
			return Decision{}, fmt.Errorf("payable %s: %w", p.ID, err)
		}
		due := nd
		for !cal.IsBusinessDay(due) {
			due = due.AddDate(0, 0, -1)
		}
		d.DueDates = append(d.DueDates, DueDate{PayableID: p.ID, NominalDue: p.NominalDue, Due: due.Format(dateLayout)})
	}
	d.Digest = DecisionDigest(d)
	return d, nil
}

// JobAt returns the scheduled instant of a job of the decision.
func (d Decision) JobAt(name string) (time.Time, error) {
	for _, j := range d.Jobs {
		if j.Name == name {
			return time.Parse(time.RFC3339, j.At)
		}
	}
	return time.Time{}, fmt.Errorf("decision has no job %q", name)
}

// Location returns the decision's timezone.
func (d Decision) Location() (*time.Location, error) { return time.LoadLocation(d.Timezone) }

// Verify recomputes the digest of a received decision.
func (d Decision) Verify() error {
	if d.Schema != DecisionSchema {
		return fmt.Errorf("decision schema %q, want %q", d.Schema, DecisionSchema)
	}
	if d.Digest == "" || d.Digest != DecisionDigest(d) {
		return fmt.Errorf("decision for %s: %w", d.TargetMonth, ErrDigestMismatch)
	}
	return nil
}

// JobShift is one scheduled job whose time changed between two decisions.
type JobShift struct {
	Name string `json:"name"`
	From string `json:"from"`
	To   string `json:"to"`
}

// DueShift is one payable whose due date changed between two decisions.
type DueShift struct {
	PayableID  string `json:"payableId"`
	NominalDue string `json:"nominalDue"`
	From       string `json:"from"`
	To         string `json:"to"`
	// DueOnCloseDate is true when the new due date is the close date of the
	// new decision: the item must be paid on the day of the close.
	DueOnCloseDate bool `json:"dueOnCloseDate"`
}

// Impact is the recomputation difference caused by a calendar revision.
type Impact struct {
	TargetMonth    string     `json:"targetMonth"`
	FromVersion    int        `json:"fromVersion"`
	ToVersion      int        `json:"toVersion"`
	FromDigest     string     `json:"fromDecisionDigest"`
	ToDigest       string     `json:"toDecisionDigest"`
	CloseDateFrom  string     `json:"closeDateFrom"`
	CloseDateTo    string     `json:"closeDateTo"`
	JobShifts      []JobShift `json:"jobShifts"`
	DueShifts      []DueShift `json:"dueShifts"`
	AddedHolidays  []string   `json:"addedHolidays"`
	RemovedHoliday []string   `json:"removedHolidays"`
}

// DiffDecisions lists every computed date that a calendar revision changed.
func DiffDecisions(from, to Decision, fromCal, toCal Calendar) Impact {
	im := Impact{TargetMonth: to.TargetMonth, FromVersion: from.CalendarVersion, ToVersion: to.CalendarVersion,
		FromDigest: from.Digest, ToDigest: to.Digest, CloseDateFrom: from.CloseDate, CloseDateTo: to.CloseDate,
		JobShifts: []JobShift{}, DueShifts: []DueShift{}, AddedHolidays: []string{}, RemovedHoliday: []string{}}
	fromJobs := map[string]string{}
	for _, j := range from.Jobs {
		fromJobs[j.Name] = j.At
	}
	for _, j := range to.Jobs {
		if f := fromJobs[j.Name]; f != j.At {
			im.JobShifts = append(im.JobShifts, JobShift{Name: j.Name, From: f, To: j.At})
		}
	}
	fromDue := map[string]string{}
	for _, d := range from.DueDates {
		fromDue[d.PayableID] = d.Due
	}
	for _, d := range to.DueDates {
		if f := fromDue[d.PayableID]; f != d.Due {
			im.DueShifts = append(im.DueShifts, DueShift{PayableID: d.PayableID, NominalDue: d.NominalDue, From: f, To: d.Due, DueOnCloseDate: d.Due == to.CloseDate})
		}
	}
	set := func(c Calendar) map[string]bool {
		m := map[string]bool{}
		for _, h := range c.Holidays {
			m[h.Date] = true
		}
		return m
	}
	a, b := set(fromCal), set(toCal)
	for d := range b {
		if !a[d] {
			im.AddedHolidays = append(im.AddedHolidays, d)
		}
	}
	for d := range a {
		if !b[d] {
			im.RemovedHoliday = append(im.RemovedHoliday, d)
		}
	}
	sort.Strings(im.AddedHolidays)
	sort.Strings(im.RemovedHoliday)
	return im
}

// BusinessDateLocal is the implemented business date rule.
const BusinessDateLocal = "local-date-of-acceptance"

// BusinessDate returns the business date and month of an acceptance instant
// under the decision's rule and timezone.
func (d Decision) BusinessDate(acceptedAt time.Time) (date, month string, err error) {
	if d.BusinessDateRule != BusinessDateLocal {
		return "", "", fmt.Errorf("business date rule %q is not implemented", d.BusinessDateRule)
	}
	loc, err := d.Location()
	if err != nil {
		return "", "", err
	}
	local := acceptedAt.In(loc)
	return local.Format(dateLayout), local.Format("2006-01"), nil
}

// NextMonth returns the month after a "YYYY-MM" month.
func NextMonth(month string) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		panic(err)
	}
	return t.AddDate(0, 1, 0).Format("2006-01")
}

func readJSON(path string, v any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeJSON(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

// ReadDecision reads and verifies a decision file.
func ReadDecision(path string) (Decision, error) {
	var d Decision
	if err := readJSON(path, &d); err != nil {
		return Decision{}, err
	}
	return d, d.Verify()
}

// WriteDecision writes a decision file.
func WriteDecision(path string, d Decision) error { return writeJSON(path, d) }

// ReadCalendar reads a calendar file and fills its digest when absent.
func ReadCalendar(path string) (Calendar, error) {
	var c Calendar
	if err := readJSON(path, &c); err != nil {
		return Calendar{}, err
	}
	if err := ValidateCalendar(c); err != nil {
		return Calendar{}, err
	}
	if c.Digest == "" {
		c.Digest = CalendarDigest(c)
	}
	return c, nil
}

// ReadRule reads a close rule file.
func ReadRule(path string) (CloseRule, error) {
	var r CloseRule
	if err := readJSON(path, &r); err != nil {
		return CloseRule{}, err
	}
	if r.Schema != RuleSchema {
		return CloseRule{}, fmt.Errorf("rule schema %q, want %q", r.Schema, RuleSchema)
	}
	return r, nil
}
