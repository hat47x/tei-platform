// First Value UX Audit reduces explicitly recorded work-session events to
// reproducible friction metrics. It never verifies app readiness or success.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

const schema = "tei.first-value-ux-session/v1"

var gitSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

type Event struct {
	At          string `json:"at"`
	Kind        string `json:"kind"`
	ExitCode    *int   `json:"exitCode,omitempty"`
	Count       int    `json:"count,omitempty"`
	EvidenceRef string `json:"evidenceRef,omitempty"`
}

type Session struct {
	Schema        string  `json:"schema"`
	SessionID     string  `json:"sessionId"`
	TaskID        string  `json:"taskId"`
	Phase         string  `json:"phase"`
	Actor         string  `json:"actor"`
	Provenance    string  `json:"provenance"`
	RepositorySHA string  `json:"repositorySHA"`
	StartedAt     string  `json:"startedAt"`
	FinishedAt    string  `json:"finishedAt,omitempty"`
	Events        []Event `json:"events"`
}

type Metrics struct {
	Schema                string   `json:"schema"`
	SessionID             string   `json:"sessionId"`
	TaskID                string   `json:"taskId"`
	Phase                 string   `json:"phase"`
	Actor                 string   `json:"actor"`
	Provenance            string   `json:"provenance"`
	RepositorySHA         string   `json:"repositorySHA"`
	DurationSeconds       *float64 `json:"durationSeconds,omitempty"`
	Commands              int      `json:"commands"`
	FailedCommands        int      `json:"failedCommands"`
	SourceEdits           int      `json:"sourceEdits"`
	ManualFields          int      `json:"manualFields"`
	RecoveryActions       int      `json:"recoveryActions"`
	IndependentReadClaims int      `json:"independentReadClaims"`
	EvidenceReferences    []string `json:"evidenceReferences"`
	Qualification         string   `json:"qualification"`
	Limitations           []string `json:"limitations"`
}

func decode(r io.Reader) (Session, error) {
	var s Session
	raw, err := io.ReadAll(io.LimitReader(r, (1<<20)+1))
	if err != nil {
		return s, fmt.Errorf("read session: %w", err)
	}
	if len(raw) > 1<<20 {
		return s, errors.New("session JSON exceeds 1 MiB limit")
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return s, fmt.Errorf("decode session: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return s, errors.New("unexpected trailing JSON value")
		}
		return s, fmt.Errorf("trailing JSON: %w", err)
	}
	return s, nil
}

func reduce(s Session) (Metrics, error) {
	out := Metrics{Schema: "tei.first-value-ux-metrics/v1", SessionID: s.SessionID, TaskID: s.TaskID, Phase: s.Phase, Actor: s.Actor, Provenance: s.Provenance, RepositorySHA: s.RepositorySHA, EvidenceReferences: []string{}, Qualification: "unverified"}
	if s.Schema != schema {
		return out, fmt.Errorf("unsupported session schema %q", s.Schema)
	}
	if strings.TrimSpace(s.SessionID) == "" || strings.TrimSpace(s.TaskID) == "" || strings.TrimSpace(s.RepositorySHA) == "" {
		return out, errors.New("sessionId, taskId and repositorySHA are required")
	}
	if !gitSHA.MatchString(s.RepositorySHA) {
		return out, errors.New("repositorySHA must be a full 40-character lowercase Git commit SHA")
	}
	if s.Phase != "E0" && s.Phase != "E1" {
		return out, fmt.Errorf("unknown phase %q", s.Phase)
	}
	if s.Actor != "new-developer" && s.Actor != "new-agent" && s.Actor != "experienced-developer" {
		return out, fmt.Errorf("unknown actor %q", s.Actor)
	}
	if s.Provenance != "observed" && s.Provenance != "synthetic" {
		return out, fmt.Errorf("unknown provenance %q", s.Provenance)
	}
	start, err := time.Parse(time.RFC3339Nano, s.StartedAt)
	if err != nil {
		return out, fmt.Errorf("startedAt: %w", err)
	}
	var end time.Time
	if s.FinishedAt != "" {
		end, err = time.Parse(time.RFC3339Nano, s.FinishedAt)
		if err != nil {
			return out, fmt.Errorf("finishedAt: %w", err)
		}
		if end.Before(start) {
			return out, errors.New("finishedAt precedes startedAt")
		}
		duration := end.Sub(start).Seconds()
		out.DurationSeconds = &duration
	} else {
		out.Limitations = append(out.Limitations, "duration-not-recorded")
	}
	if len(s.Events) == 0 {
		return out, errors.New("events must not be empty")
	}
	references := map[string]struct{}{}
	var previous time.Time
	for i, e := range s.Events {
		at, err := time.Parse(time.RFC3339Nano, e.At)
		if err != nil {
			return out, fmt.Errorf("events[%d].at: %w", i, err)
		}
		if at.Before(start) || (!end.IsZero() && at.After(end)) {
			return out, fmt.Errorf("events[%d] outside session time window", i)
		}
		if i > 0 && at.Before(previous) {
			return out, fmt.Errorf("events[%d] timestamp out of order", i)
		}
		previous = at
		switch e.Kind {
		case "command":
			if e.ExitCode == nil || e.Count != 0 {
				return out, fmt.Errorf("events[%d] command requires exitCode and no count", i)
			}
			out.Commands++
			if *e.ExitCode != 0 {
				out.FailedCommands++
			}
		case "source-edit":
			if e.ExitCode != nil || e.Count != 0 {
				return out, fmt.Errorf("events[%d] source-edit has invalid fields", i)
			}
			out.SourceEdits++
		case "manual-fields":
			if e.ExitCode != nil || e.Count < 1 {
				return out, fmt.Errorf("events[%d] manual-fields requires count > 0", i)
			}
			out.ManualFields += e.Count
		case "recovery":
			if e.ExitCode != nil || e.Count != 0 {
				return out, fmt.Errorf("events[%d] recovery has invalid fields", i)
			}
			out.RecoveryActions++
		case "independent-read":
			if e.ExitCode != nil || e.Count != 0 || strings.TrimSpace(e.EvidenceRef) == "" {
				return out, fmt.Errorf("events[%d] independent-read requires evidenceRef", i)
			}
			if s.Phase != "E1" {
				return out, fmt.Errorf("events[%d] independent-read cannot establish E0", i)
			}
			out.IndependentReadClaims++
		default:
			return out, fmt.Errorf("events[%d] unknown kind %q", i, e.Kind)
		}
		if e.EvidenceRef != "" {
			references[e.EvidenceRef] = struct{}{}
		}
	}
	for ref := range references {
		out.EvidenceReferences = append(out.EvidenceReferences, ref)
	}
	sort.Strings(out.EvidenceReferences)
	if s.Provenance == "synthetic" {
		out.Limitations = append(out.Limitations, "synthetic-session-not-user-evidence")
	}
	if s.Phase == "E1" && out.IndependentReadClaims == 0 {
		out.Limitations = append(out.Limitations, "independent-read-not-recorded")
	}
	out.Limitations = append(out.Limitations, "event-claims-and-evidence-references-not-independently-verified")
	return out, nil
}

func execute(r io.Reader, w io.Writer) error {
	s, err := decode(r)
	if err != nil {
		return err
	}
	result, err := reduce(s)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func main() {
	input := flag.String("input", "", "session JSON file (read-only)")
	flag.Parse()
	if *input == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: first-value-ux-audit -input session.json")
		os.Exit(2)
	}
	f, err := os.Open(*input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer f.Close()
	if err = execute(f, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
