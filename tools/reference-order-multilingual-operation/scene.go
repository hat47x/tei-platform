package ordermultilingualoperation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/plugin"
	localization "tei/reference/message-localization"
	overlays "tei/reference/multilingual-overlays"
	messages "tei/reference/standard-messages"
)

// Scene is one state of the operation: one workspace load and lock, one
// source release, the confirmed translations, the host grants and the host.
type Scene struct {
	U         Unified
	WS        Workspace
	Rel       SourceRelease
	Conf      Confirmed
	Grants    Grants
	Host      *Host
	Providers []plugin.Identity
}

func (s Scene) messageRequest(ctx context.Context, id string, policy localization.Policy, records []localization.TranslationRecord) localization.WorkspaceRequest {
	stores, bindings := s.U.StoresAndBindings(s.Providers)
	return localization.WorkspaceRequest{
		Root: s.WS.Root, Environment: s.U.Environment, Lock: s.WS.Lock,
		Messages: s.U.Catalog(s.Rel), Stores: stores, Bindings: bindings, Records: records,
		Grants: s.Grants.Messages, Policy: policy,
		Request: localization.Request{Application: s.U.Application, Message: s.U.MessageRef(id)},
		Review:  s.Host.MessageGate(ctx, s.WS, s.Rel),
	}
}

// ResolveMessage selects a message for a locale through
// reference/message-localization with every current record.
func (s Scene) ResolveMessage(ctx context.Context, id, locale string) (localization.Output, error) {
	return localization.ResolveWorkspace(s.messageRequest(ctx, id, localization.Policy{Locale: locale}, s.U.MessageRecords(s.Conf, nil)))
}

// BuildBundle builds the locale bundle candidate of the owner's displays
// after the host check.
func (s Scene) BuildBundle(ctx context.Context, locales []string) (overlays.LocaleBundle, error) {
	return s.buildBundle(ctx, locales, s.U.Packs(s.Conf, nil), s.Grants.Displays)
}

func (s Scene) buildBundle(ctx context.Context, locales []string, packs []overlays.Pack, grants []overlays.Grant) (overlays.LocaleBundle, error) {
	if err := s.Host.Check(ctx, s.WS, s.Rel); err != nil {
		return overlays.LocaleBundle{}, err
	}
	graph, err := s.U.Graph(s.WS, s.Rel)
	if err != nil {
		return overlays.LocaleBundle{}, err
	}
	return overlays.BuildLockedLocaleBundle(s.WS.Catalog, s.WS.Env, s.WS.Lock, graph, s.U.Surfaces(s.Rel), packs, grants, nil,
		overlays.LocaleBundleRequest{Target: s.U.Owner, Keys: s.U.DisplayKeys(s.Rel), Locales: locales})
}

// Shown is one text the business screen shows.
type Shown struct {
	Kind      string `json:"kind"`
	Target    string `json:"target"`
	Requested string `json:"requested"`
	Locale    string `json:"locale"`
	Text      string `json:"text"`
	Path      string `json:"path"` // translated, source, source-only, source-direct
	Refusal   string `json:"refusal,omitempty"`
}

// Show renders every message and display for the requested locale. When the
// translated path is refused, it switches explicitly to the source language
// (message policy source-only, a bundle without packs) and records the
// refusal; when even that is refused (a stale lock), it shows the source
// text of the accepted release directly. It never falls back to a stored
// translation.
func (s Scene) Show(ctx context.Context, locale string) []Shown {
	var out []Shown
	for _, m := range s.Rel.Messages {
		sh := Shown{Kind: "message", Target: m.ID, Requested: locale}
		res, err := s.ResolveMessage(ctx, m.ID, locale)
		if err == nil {
			sh.Locale, sh.Text, sh.Path = res.Resource.Locale, res.Resource.Text, "translated"
			if res.Resource.FromSource {
				sh.Path = "source"
			}
		} else {
			sh.Refusal = err.Error()
			src, err2 := localization.ResolveWorkspace(s.messageRequest(ctx, m.ID,
				localization.Policy{Mode: "source-only", Locale: m.SourceLocale}, nil))
			if err2 == nil {
				sh.Locale, sh.Text, sh.Path = src.Resource.Locale, src.Resource.Text, "source-only"
			} else {
				def, err3 := s.U.Catalog(s.Rel).Lookup(s.U.MessageRef(m.ID))
				if err3 == nil {
					sh.Locale, sh.Text, sh.Path = def.SourceLocale, def.Text, "source-direct"
					sh.Refusal += " | source-only: " + err2.Error()
				}
			}
		}
		out = append(out, sh)
	}
	bundle, err := s.BuildBundle(ctx, []string{locale})
	path := "translated"
	refusal := ""
	if locale == s.U.SourceLocale {
		path = "source"
	}
	if err != nil {
		refusal = err.Error()
		bundle, err = s.buildBundle(ctx, []string{s.U.SourceLocale}, nil, nil)
		path = "source-only"
		if err != nil {
			refusal += " | source-only: " + err.Error()
			path = "source-direct"
		}
	}
	for _, d := range s.Rel.Displays {
		kind := "display-name"
		if d.Kind == "capability" {
			kind = "function-name"
		}
		sh := Shown{Kind: kind, Target: DisplayTarget(d), Requested: locale, Path: path, Refusal: refusal}
		if path == "source-direct" {
			sh.Locale, sh.Text = s.U.SourceLocale, d.Text
		} else {
			key := s.U.ResourceKey(DisplayTarget(d))
			for _, dm := range bundle.Displays {
				if dm.Resource == key {
					sh.Locale, sh.Text = dm.Locale, dm.Text
				}
			}
		}
		out = append(out, sh)
	}
	return out
}

// Cell is one observed state of a matrix row.
type Cell struct {
	Status     string `json:"status"` // valid, expired, refused, absent
	Diagnostic string `json:"diagnostic,omitempty"`
	Text       string `json:"text,omitempty"`
	BasedOn    string `json:"basedOn,omitempty"`
}

// MatrixRow is a row of the reconciliation table of the same target list
// before and after the source revision.
type MatrixRow struct {
	Kind          string `json:"kind"`
	Target        string `json:"target"`
	Locale        string `json:"locale"`
	Provider      string `json:"provider"`
	Category      string `json:"category"`
	Expected      string `json:"expectedAfter"`
	NativeBefore  string `json:"nativeRevisionBefore"`
	NativeAfter   string `json:"nativeRevisionAfter"`
	SourceChanged bool   `json:"sourceChanged"`
	RuleAfter     string `json:"ruleAfter"`
	Before        Cell   `json:"before"`
	After         Cell   `json:"after"`
	Agrees        bool   `json:"agrees"`
}

func (r MatrixRow) slotKind() string {
	if r.Kind == "message" {
		return "message"
	}
	return "display"
}

func (s Scene) confirmedFor(r MatrixRow) (ConfirmedTranslation, bool) {
	for _, t := range s.Conf.Current {
		if t.Kind == r.slotKind() && t.Target == r.Target && t.Locale == r.Locale && t.Provider == r.Provider {
			return t, true
		}
	}
	return ConfirmedTranslation{}, false
}

func cellFromErr(err error) Cell {
	msg := err.Error()
	if strings.Contains(msg, "stale-translation") {
		return Cell{Status: "expired", Diagnostic: msg}
	}
	return Cell{Status: "refused", Diagnostic: msg}
}

// Evaluate observes one row through the existing assets: a message row
// through message-localization with only that row's record, a display row
// through BuildLockedDisplay with only that row's pack.
func (s Scene) Evaluate(ctx context.Context, r MatrixRow) Cell {
	if r.slotKind() == "message" {
		var records []localization.TranslationRecord
		var based string
		if r.Provider != "source" {
			t, ok := s.confirmedFor(r)
			if !ok {
				return Cell{Status: "absent"}
			}
			based = t.BasedOn
			records = s.U.MessageRecords(Confirmed{Current: []ConfirmedTranslation{t}}, nil)
		}
		out, err := localization.ResolveWorkspace(s.messageRequest(ctx, r.Target, localization.Policy{Locale: r.Locale}, records))
		if err != nil {
			c := cellFromErr(err)
			c.BasedOn = based
			return c
		}
		if out.Resource.Locale != r.Locale || out.Resource.FromSource != (r.Provider == "source") {
			return Cell{Status: "refused", Diagnostic: "unexpected selection " + out.Resource.Locale, BasedOn: based}
		}
		return Cell{Status: "valid", Text: out.Resource.Text, BasedOn: based}
	}
	var packs []overlays.Pack
	var based string
	if r.Provider != "source" {
		t, ok := s.confirmedFor(r)
		if !ok {
			return Cell{Status: "absent"}
		}
		based = t.BasedOn
		packs = s.U.Packs(Confirmed{Current: []ConfirmedTranslation{t}}, nil)
	}
	if err := s.Host.Check(ctx, s.WS, s.Rel); err != nil {
		return Cell{Status: "refused", Diagnostic: err.Error(), BasedOn: based}
	}
	graph, err := s.U.Graph(s.WS, s.Rel)
	if err != nil {
		return Cell{Status: "refused", Diagnostic: err.Error(), BasedOn: based}
	}
	key := s.U.ResourceKey(r.Target)
	m, err := overlays.BuildLockedDisplay(s.WS.Catalog, s.WS.Env, s.WS.Lock, graph, s.U.Surfaces(s.Rel), packs,
		s.Grants.Displays, nil, overlays.Lookup{Target: s.U.Owner, Key: key, Locale: r.Locale})
	if err != nil {
		c := cellFromErr(err)
		c.BasedOn = based
		return c
	}
	if m.Locale != r.Locale || m.FromSource != (r.Provider == "source") {
		return Cell{Status: "refused", Diagnostic: "unexpected selection " + m.Locale, BasedOn: based}
	}
	return Cell{Status: "valid", Text: m.Text, BasedOn: based}
}

// CheckBundleAgainstSources lists the displays of a bundle whose source
// revision is not the current source of the release. This is the sample's
// host side check; the asset checks only integrity and the lock.
func (u Unified) CheckBundleAgainstSources(b overlays.LocaleBundle, rel SourceRelease) []string {
	current := map[string]string{}
	for _, d := range rel.Displays {
		k := u.ResourceKey(DisplayTarget(d))
		current[keyString(k)] = overlays.Fingerprint(d.Text)
	}
	var stale []string
	for _, d := range b.Displays {
		if current[keyString(d.Resource)] != d.SourceRevision {
			stale = append(stale, d.Locale+" "+d.Resource.Target.Subject.ID)
		}
	}
	sort.Strings(stale)
	return stale
}

func keyString(k interface{}) string {
	b, _ := json.Marshal(k)
	return string(b)
}

// ForgeDigest recomputes a bundle digest the same way the asset does
// (SHA-256 of the Go JSON without the digest). It exists to show that a
// rewritten text with a recomputed digest passes the integrity check.
func ForgeDigest(b overlays.LocaleBundle) string {
	b.Digest = ""
	payload, _ := json.Marshal(b)
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Admission is the host's decision on a display pinned to an expected bundle
// digest.
type Admission struct {
	PinnedDigest      string   `json:"pinnedDigest"`
	ReceivedDigest    string   `json:"receivedDigest"`
	Integrity         string   `json:"integrity"`
	LockCheck         string   `json:"lockCheck"`
	StaleSources      []string `json:"staleSources,omitempty"`
	RegeneratedDigest string   `json:"regeneratedDigest,omitempty"`
	Regenerate        string   `json:"regenerate"`
	Admitted          bool     `json:"admitted"`
	Reason            string   `json:"reason,omitempty"`
}

func errText(err error) string {
	if err == nil {
		return "ok"
	}
	return err.Error()
}

// Admit decides whether a received bundle may be shown for a display pinned
// to pinned. Integrity and the lock are the asset's checks; the source check
// and the regeneration (the trusted expected digest from another path) are
// the sample's.
func (s Scene) Admit(ctx context.Context, pinned string, received overlays.LocaleBundle) Admission {
	a := Admission{PinnedDigest: pinned, ReceivedDigest: received.Digest}
	a.Integrity = errText(overlays.CheckLocaleBundleIntegrity(received))
	a.LockCheck = errText(overlays.CheckLocaleBundleAgainstLock(received, s.WS.Catalog, s.WS.Env, s.WS.Lock))
	a.StaleSources = s.U.CheckBundleAgainstSources(received, s.Rel)
	regen, err := s.BuildBundle(ctx, received.Locales)
	a.Regenerate = errText(err)
	if err == nil {
		a.RegeneratedDigest = regen.Digest
	}
	switch {
	case a.Integrity != "ok":
		a.Reason = "integrity: " + a.Integrity
	case a.LockCheck != "ok":
		a.Reason = "lock: " + a.LockCheck
	case len(a.StaleSources) > 0:
		a.Reason = fmt.Sprintf("stale sources: %v", a.StaleSources)
	case a.Regenerate != "ok":
		a.Reason = "regeneration refused: " + a.Regenerate
	case received.Digest != pinned:
		a.Reason = "received digest is not the pinned digest"
	case a.RegeneratedDigest != pinned:
		a.Reason = "pinned digest is not the current regenerated candidate"
	default:
		a.Admitted = true
	}
	return a
}

// EnglishTexts is every English translation text known in any release
// (current and history), used to check that none is shown silently.
func (c Confirmed) EnglishTexts() map[string]bool {
	out := map[string]bool{}
	for _, list := range [][]ConfirmedTranslation{c.Current, c.History} {
		for _, t := range list {
			if strings.HasPrefix(t.Locale, "en") {
				out[t.Text] = true
				for _, v := range t.Plural {
					out[v] = true
				}
			}
		}
	}
	return out
}

// definition looks up a message definition of the release.
func (s Scene) definition(id string) (messages.Definition, error) {
	return s.U.Catalog(s.Rel).Lookup(s.U.MessageRef(id))
}
