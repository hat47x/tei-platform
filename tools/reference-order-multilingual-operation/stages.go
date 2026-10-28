package ordermultilingualoperation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/modelref"
	"tei/internal/plugin"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	localization "tei/reference/message-localization"
	overlays "tei/reference/multilingual-overlays"
	messages "tei/reference/standard-messages"
)

// Chain holds the running state of the through run.
type Chain struct {
	Repo   string // repository root (reference assets are read from here)
	Work   string // scratch directory (workspace copy)
	U      Unified
	Rec    Record
	Host   *Host
	Trans  []Translation
	Rel1   SourceRelease
	Rel2   SourceRelease
	WS1    Workspace
	WS2    Workspace
	S2     Scene
	S4     Scene
	B1     overlays.LocaleBundle
	B2     overlays.LocaleBundle
	Matrix []MatrixRow
}

func (c *Chain) testdata(name string) string {
	return filepath.Join(c.Repo, "tools", "reference-order-multilingual-operation", "testdata", name)
}

func (c *Chain) wsRoot() string { return filepath.Join(c.Work, "workspace") }

// NewChain prepares a chain over the repository root and a scratch dir.
func NewChain(repo, work string) *Chain {
	u := DefaultUnified()
	return &Chain{Repo: repo, Work: work, U: u, Host: NewHost(u),
		Rec: Record{Schema: "tei.reference.order-multilingual-operation-record/v0", Unified: u}}
}

func refused(err error, want string) bool { return err != nil && strings.Contains(err.Error(), want) }

// Stage1 copies the workspace, adds the ordinary owner plugin and
// std.message (no language plugin), validates the message catalog and
// resolves every message in its source language only.
func (c *Chain) Stage1(ctx context.Context) error {
	u := c.U
	if err := CopyWorkspace(filepath.Join(c.Repo, "reference", "workspace"), c.wsRoot()); err != nil {
		return err
	}
	if err := WritePlugin(c.wsRoot(), u.Message, PluginSource(u.Message)); err != nil {
		return err
	}
	if err := WritePlugin(c.wsRoot(), u.Owner, PluginSource(u.Owner, string(u.Message.ID))); err != nil {
		return err
	}
	if err := SetEnvironmentPlugin(c.wsRoot(), u.Environment, u.Owner.ID, true); err != nil {
		return err
	}
	ws, err := Open(c.wsRoot(), u)
	if err != nil {
		return fmt.Errorf("stage 1: %w", err)
	}
	c.WS1 = ws
	if c.Trans, err = LoadTranslations(c.testdata("translations.json")); err != nil {
		return err
	}
	if c.Rel1, err = LoadRelease(c.testdata("source-r1.json")); err != nil {
		return err
	}
	if c.Rel2, err = LoadRelease(c.testdata("source-r2.json")); err != nil {
		return err
	}
	// The order business definition stays byte identical to the baseline.
	orig, err := os.ReadFile(filepath.Join(c.Repo, "reference", "workspace", "semantic", "sales", "order.tei"))
	if err != nil {
		return err
	}
	var orderFP string
	for _, a := range ws.Index.Artifacts {
		if a.Path == "semantic/sales/order.tei" {
			orderFP = a.Fingerprint
		}
	}
	typed := map[string]bool{}
	for _, d := range c.Rel1.Displays {
		typed[DisplayTarget(d)] = ws.HasNode(u.SubjectKey(d.Kind, d.ID))
	}
	catalog := u.Catalog(c.Rel1)
	validateErr := catalog.Validate()
	dup := messages.Catalog{Definitions: append(append([]messages.Definition(nil), catalog.Definitions...), catalog.Definitions[0])}
	_, missingErr := catalog.Lookup(u.MessageRef("order.not_declared"))
	m1, _ := catalog.Lookup(u.MessageRef("order.batch_confirmed"))
	typeErr := messages.ValidateArguments(m1, map[string]messages.ParameterKind{"count": messages.ParameterText})
	okArgs := messages.ValidateArguments(m1, map[string]messages.ParameterKind{"count": messages.ParameterInteger})
	revisions := map[string]string{}
	for _, d := range catalog.Definitions {
		revisions[string(d.Ref.ID)] = messages.Revision(d)
	}
	c.Host.Accept(ws, c.Rel1, "synthetic-host-decision-r1-without-language")
	s := Scene{U: u, WS: ws, Rel: c.Rel1, Host: c.Host}
	source := map[string]string{}
	var sourceErrs []string
	for _, m := range c.Rel1.Messages {
		out, err := s.ResolveMessage(ctx, m.ID, m.SourceLocale)
		if err != nil || !out.Resource.FromSource || out.Resource.Locale != m.SourceLocale {
			sourceErrs = append(sourceErrs, fmt.Sprintf("%s: %v", m.ID, err))
			continue
		}
		source[m.ID] = out.Resource.Text
	}
	_, enErr := s.ResolveMessage(ctx, "order.batch_confirmed", "en-US")
	resolved, _ := ws.Resolved()
	ok := validateErr == nil && refused(dup.Validate(), "duplicate-message") && refused(missingErr, "missing-message") &&
		refused(typeErr, "argument-type-mismatch") && okArgs == nil && len(sourceErrs) == 0 && enErr != nil &&
		orderFP != "" && orderFP == sourceindex.Fingerprint(orig)
	for _, v := range typed {
		ok = ok && v
	}
	c.Rec.AddStage(StageRecord{Stage: 1, Name: "message source definition", LockFP: ws.LockFP,
		Release: c.Rel1.Release, ReleaseHash: c.Rel1.Digest,
		Assets: []AssetCall{{"reference/standard-messages", "Catalog.Validate, Lookup, Revision, ValidateArguments"},
			{"reference/message-localization", "ResolveWorkspace (source only, no language plugin)"},
			{"reference/workspace", "copied; std.message and sales.order_entry added"}},
		Outputs: map[string]any{
			"passed": ok, "modelRevision": ws.ModelRevision, "resolvedPlugins": resolved, "lockedPlugins": sortedKeys(LockedEntries(ws.Lock, u.Environment)),
			"messageRevisions": revisions, "sourceResolution": source, "sourceResolutionErrors": sourceErrs,
			"englishWithoutLanguagePlugin": errText(enErr), "typedTargetsInDerivedIndex": typed,
			"catalogValidate": errText(validateErr), "duplicateRefused": errText(dup.Validate()),
			"undeclaredRefused": errText(missingErr), "argumentTypeRefused": errText(typeErr),
			"orderDefinitionFingerprint": orderFP, "orderDefinitionBaselineFingerprint": sourceindex.Fingerprint(orig),
		}})
	if !ok {
		return fmt.Errorf("stage 1 did not hold: %v %v", sourceErrs, typed)
	}
	return nil
}

// Stage2 adds the language plugins to the environment, confirms the r1
// translations and selects every message in ja-JP, en-US and de-DE.
func (c *Chain) Stage2(ctx context.Context) error {
	u := c.U
	for _, p := range Languages {
		if err := WritePlugin(c.wsRoot(), p, PluginSource(p, string(u.Message.ID))); err != nil {
			return err
		}
		if err := SetEnvironmentPlugin(c.wsRoot(), u.Environment, p.ID, true); err != nil {
			return err
		}
	}
	ws, err := Open(c.wsRoot(), u)
	if err != nil {
		return fmt.Errorf("stage 2: %w", err)
	}
	c.WS2 = ws
	before, after := LockedEntries(c.WS1.Lock, u.Environment), LockedEntries(ws.Lock, u.Environment)
	unchanged := true
	for k, v := range before {
		if after[k] != v {
			unchanged = false
		}
	}
	var added []string
	for _, k := range sortedKeys(after) {
		if _, ok := before[k]; !ok {
			added = append(added, k)
		}
	}
	conf, err := u.Confirm(Confirmed{}, c.Rel1, c.Trans)
	if err != nil {
		return err
	}
	c.Host.Accept(ws, c.Rel1, "synthetic-host-decision-r1-with-language")
	c.S2 = Scene{U: u, WS: ws, Rel: c.Rel1, Conf: conf, Grants: u.GrantAll(conf), Host: c.Host, Providers: Languages}
	selection := map[string]map[string]string{}
	for _, m := range c.Rel1.Messages {
		selection[m.ID] = map[string]string{}
		for _, loc := range []string{"ja-JP", "en-US", "de-DE"} {
			out, err := c.S2.ResolveMessage(ctx, m.ID, loc)
			if err != nil {
				selection[m.ID][loc] = "refused: " + err.Error()
			} else if out.Resource.FromSource {
				selection[m.ID][loc] = "source: " + out.Resource.Text
			} else {
				selection[m.ID][loc] = "translated: " + out.Resource.Text
			}
		}
	}
	ok := unchanged && len(added) == len(Languages) &&
		strings.HasPrefix(selection["order.batch_confirmed"]["en-US"], "translated: Confirmed {count} orders.") &&
		strings.HasPrefix(selection["order.batch_confirmed"]["ja-JP"], "source: ") &&
		strings.HasPrefix(selection["order.submit_rejected"]["en-US"], "translated: ") &&
		strings.HasPrefix(selection["order.tax_notice"]["en-US"], "source: ") &&
		strings.HasPrefix(selection["order.tax_notice"]["de-DE"], "refused: ")
	c.Rec.AddStage(StageRecord{Stage: 2, Name: "english translation retrofit", LockFP: ws.LockFP,
		Release: c.Rel1.Release, ReleaseHash: c.Rel1.Digest,
		Assets: []AssetCall{{"reference/message-localization", "ResolveWorkspace (stores, bindings, grants, host review gate)"},
			{"reference/multilingual-resources", "Resolve (inside message-localization)"}},
		Outputs: map[string]any{"passed": ok, "ordinaryLockEntriesUnchanged": unchanged, "addedLockEntries": added,
			"selection": selection, "confirmedTranslations": len(conf.Current), "messageGrants": len(c.S2.Grants.Messages)}})
	if !ok {
		return fmt.Errorf("stage 2 did not hold: %v", selection)
	}
	return nil
}

// Stage3 builds the en-US/ja-JP bundle of display and function names in the
// same environment and lock and pins its digest.
func (c *Chain) Stage3(ctx context.Context) error {
	u := c.U
	b, err := c.S2.BuildBundle(ctx, []string{"en-US", "ja-JP"})
	if err != nil {
		return fmt.Errorf("stage 3 bundle: %w", err)
	}
	c.B1 = b
	admit := c.S2.Admit(ctx, b.Digest, b)
	displays := map[string]string{}
	for _, d := range b.Displays {
		displays[d.Locale+" "+d.Resource.Target.Subject.ID] = d.Text
	}
	messagesEN := map[string]string{}
	for _, m := range c.Rel1.Messages {
		out, err := c.S2.ResolveMessage(ctx, m.ID, "en-US")
		if err == nil {
			messagesEN[m.ID] = out.Resource.Text
		}
	}
	// The overlay's workspace path takes the source from the Symbol of the
	// data declaration; called once to record what it yields.
	_, witness, wsErr := overlays.BuildAcceptedWorkspaceDisplay(ctx, c.Host, string(u.Application), c.WS2.Root, c.WS2.Lock,
		overlays.WorkspaceDisplayConfig{Environment: u.Environment, Scope: u.Scope, Data: "sales.order", Owner: u.Owner,
			SourceLocale: u.SourceLocale, Locale: u.SourceLocale},
		func(overlays.AcceptedDisplayReview) error { return c.Host.Check(ctx, c.WS2, c.Rel1) })
	candidate := digestJSON(map[string]any{"bundle": b.Digest, "messages": messagesEN})
	ok := admit.Admitted && len(b.Displays) == 6 && displays["en-US sales.order.submit"] == "Confirm order" &&
		displays["ja-JP sales.order"] == "受注" && len(messagesEN) == 3
	c.Rec.AddStage(StageRecord{Stage: 3, Name: "display and function names, locked bundle", LockFP: c.WS2.LockFP,
		Release: c.Rel1.Release, ReleaseHash: c.Rel1.Digest,
		Assets: []AssetCall{{"reference/multilingual-overlays", "BuildLockedLocaleBundle, CheckLocaleBundleIntegrity, CheckLocaleBundleAgainstLock"},
			{"reference/multilingual-overlays", "BuildAcceptedWorkspaceDisplay (diagnostic)"},
			{"internal/sourceacceptance", "VerifyCurrent (host check)"}},
		Outputs: map[string]any{"passed": ok, "bundleDigest": b.Digest, "bundleLockFingerprint": b.LockFingerprint,
			"displays": displays, "messagesEnUS": messagesEN, "candidateDigest": candidate, "admission": admit,
			"workspaceDisplayOriginal": witness.Original, "workspaceDisplayError": errText(wsErr)}})
	if !ok {
		return fmt.Errorf("stage 3 did not hold: %+v %v", admit, displays)
	}
	return nil
}

// snapshotFiles saves the bytes of the given copy files for restoring.
func snapshotFiles(paths ...string) (func() error, error) {
	saved := map[string][]byte{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		saved[p] = b
	}
	return func() error {
		for p, b := range saved {
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(p, b, 0o644); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

// FaultBundleTampering injects the stage 3/4 specific faults: a rewritten
// text, a rewritten text with a recomputed digest, and a swapped lock.
func (c *Chain) FaultBundleTampering(ctx context.Context) error {
	u := c.U
	rewritten := c.B1
	rewritten.Displays = append([]overlays.DisplayManifest(nil), c.B1.Displays...)
	for i, d := range rewritten.Displays {
		if d.Locale == "en-US" && d.Resource.Target.Subject.ID == "sales.order.submit" {
			rewritten.Displays[i].Text = "Cancel order"
		}
	}
	plain := c.S2.Admit(ctx, c.B1.Digest, rewritten)
	forged := rewritten
	forged.Digest = ForgeDigest(forged)
	forgedAdmit := c.S2.Admit(ctx, c.B1.Digest, forged)
	c.Rec.fault("bundle-text-rewritten", "固有", "3-4", !plain.Admitted && strings.Contains(plain.Integrity, "digest-mismatch"), plain)
	c.Rec.fault("bundle-text-rewritten-digest-recomputed", "固有", "3-4",
		!forgedAdmit.Admitted && forgedAdmit.Integrity == "ok" && forgedAdmit.LockCheck == "ok" &&
			forgedAdmit.RegeneratedDigest == c.B1.Digest && forgedAdmit.ReceivedDigest != c.B1.Digest, forgedAdmit)

	// Swap the environment lock: the English plugin's definition changes
	// after the candidate was made, and the host relocks.
	enPath := pluginPath(c.wsRoot(), LangEN)
	restore, err := snapshotFiles(enPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(enPath, []byte(strings.Replace(PluginSource(LangEN, string(u.Message.ID)), "runtime builtin", "runtime builtin\n    requires plugin sales.order_entry ^1", 1)), 0o644); err != nil {
		return err
	}
	swapped, err := Open(c.wsRoot(), u)
	if err != nil {
		return err
	}
	c.Host.Accept(swapped, c.Rel1, "synthetic-host-decision-relocked")
	ss := c.S2
	ss.WS = swapped
	swapAdmit := ss.Admit(ctx, c.B1.Digest, c.B1)
	regen, regenErr := ss.BuildBundle(ctx, []string{"en-US", "ja-JP"})
	if err := restore(); err != nil {
		return err
	}
	back, err := Open(c.wsRoot(), u)
	if err != nil {
		return err
	}
	c.Host.Accept(back, c.Rel1, "synthetic-host-decision-r1-with-language")
	restored := back.LockFP == c.WS2.LockFP && back.ModelRevision == c.WS2.ModelRevision
	c.Rec.fault("environment-lock-swapped", "固有", "3-4",
		!swapAdmit.Admitted && strings.Contains(swapAdmit.LockCheck, "environment-lock-mismatch") && regenErr == nil &&
			regen.Digest != c.B1.Digest && swapped.LockFP != c.WS2.LockFP && restored,
		map[string]any{"admission": swapAdmit, "lockBefore": c.WS2.LockFP, "lockSwapped": swapped.LockFP,
			"regeneratedDigest": regen.Digest, "regenerate": errText(regenErr), "restored": restored})
	return nil
}

// pdb02Case evaluates one P-DB-02 case on a scene.
func (c *Chain) pdb02Case(ctx context.Context, id string, s Scene, forbidden map[string]bool, wantRefusal string) {
	shown := s.Show(ctx, "en-US")
	continued, silent, refusedSeen := true, []string{}, false
	for _, sh := range shown {
		if forbidden[sh.Text] {
			silent = append(silent, sh.Target+": "+sh.Text)
		}
		if sh.Text == "" {
			continued = false
		}
		if strings.Contains(sh.Refusal, wantRefusal) {
			refusedSeen = true
		}
		// Every Japanese-sourced item that was refused shows its Japanese source.
		if sh.Refusal != "" && sh.Target != "order.tax_notice" && sh.Locale != c.U.SourceLocale {
			continued = false
		}
	}
	admit := s.Admit(ctx, c.B1.Digest, c.B1)
	c.Rec.fault("pdb02-"+id, "P-DB-02", "2-3", continued && len(silent) == 0 && refusedSeen && !admit.Admitted,
		map[string]any{"shown": shown, "silentOldTranslations": silent, "pinnedAdmission": admit})
}

// FaultPDB02 revokes grants and acceptance, expires the lock, removes the
// English plugin, and checks another environment.
func (c *Chain) FaultPDB02(ctx context.Context) error {
	u := c.U
	allEnglish := c.S2.Conf.EnglishTexts()
	// (1) grants revoked
	s := c.S2
	s.Grants = c.S2.Grants.Without("en-US")
	c.pdb02Case(ctx, "grant-revoked", s, allEnglish, "not-granted")
	// (2) acceptance revoked
	c.Host.Revoke()
	c.pdb02Case(ctx, "acceptance-revoked", c.S2, allEnglish, "SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE")
	c.Host.Accept(c.WS2, c.Rel1, "synthetic-host-decision-r1-with-language")
	// (3) lock expired: the English plugin changes but the lock is the old one.
	enPath := pluginPath(c.wsRoot(), LangEN)
	restore, err := snapshotFiles(enPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(enPath, []byte(strings.Replace(PluginSource(LangEN, string(u.Message.ID)), "runtime builtin", "runtime builtin\n    requires plugin sales.order_entry ^1", 1)), 0o644); err != nil {
		return err
	}
	stale, err := Reopen(c.wsRoot(), c.WS2.Lock, u)
	if err != nil {
		return err
	}
	c.Host.Accept(stale, c.Rel1, "synthetic-host-decision-source-changed-lock-kept")
	s = c.S2
	s.WS = stale
	c.pdb02Case(ctx, "lock-expired", s, allEnglish, "fingerprint diverged")
	if err := restore(); err != nil {
		return err
	}
	// (4) English plugin removed from the environment.
	envPath := filepath.Join(c.wsRoot(), "environments", string(u.Environment), "environment.tei")
	restoreEnv, err := snapshotFiles(envPath)
	if err != nil {
		return err
	}
	if err := SetEnvironmentPlugin(c.wsRoot(), u.Environment, LangEN.ID, false); err != nil {
		return err
	}
	removed, err := Open(c.wsRoot(), u)
	if err != nil {
		return err
	}
	c.Host.Accept(removed, c.Rel1, "synthetic-host-decision-english-removed")
	s = c.S2
	s.WS = removed
	langEN := map[string]bool{}
	for _, t := range append(append([]ConfirmedTranslation(nil), c.S2.Conf.Current...), c.S2.Conf.History...) {
		if t.Provider == string(LangEN.ID) {
			langEN[t.Text] = true
		}
	}
	c.pdb02Case(ctx, "english-plugin-removed", s, langEN, "unselected")
	if err := restoreEnv(); err != nil {
		return err
	}
	back, err := Open(c.wsRoot(), u)
	if err != nil {
		return err
	}
	if back.LockFP != c.WS2.LockFP || back.ModelRevision != c.WS2.ModelRevision {
		return fmt.Errorf("P-DB-02 restore did not return to the stage 2 lock and source")
	}
	c.Host.Accept(back, c.Rel1, "synthetic-host-decision-r1-with-language")
	// (5) another environment: production has no language plugin and no grant.
	req := c.S2.messageRequest(ctx, "order.batch_confirmed", localization.Policy{Locale: "en-US"}, u.MessageRecords(c.S2.Conf, nil))
	req.Environment = "production"
	_, prodErr := localization.ResolveWorkspace(req)
	c.Rec.fault("pdb02-other-environment", "P-DB-02", "2-3", prodErr != nil, map[string]any{"production": errText(prodErr)})
	return nil
}

// LoadExpectedMatrix reads the declared target list.
func LoadExpectedMatrix(path string) ([]MatrixRow, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Rows []struct {
			Kind, Target, Locale, Provider, Category, After string
		} `json:"rows"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	var out []MatrixRow
	for _, r := range doc.Rows {
		out = append(out, MatrixRow{Kind: r.Kind, Target: r.Target, Locale: r.Locale, Provider: r.Provider,
			Category: r.Category, Expected: r.After})
	}
	return out, nil
}

// Stage4 revises the Japanese source once, reconciles the same target list
// before and after, refuses the mixed display, re-confirms the dependent
// English translations and regenerates the bundle.
func (c *Chain) Stage4(ctx context.Context) error {
	u := c.U
	rows, err := LoadExpectedMatrix(c.testdata("expected-matrix.json"))
	if err != nil {
		return err
	}
	c.Host.Accept(c.WS2, c.Rel2, "synthetic-host-decision-r2")
	pre := c.S2
	pre.Rel = c.Rel2
	before := c.S2
	// The before column is observed under the r1 acceptance.
	for i := range rows {
		kind := rows[i].slotKind()
		rows[i].NativeBefore, _ = u.NativeRevision(c.Rel1, kind, rows[i].Target)
		rows[i].NativeAfter, _ = u.NativeRevision(c.Rel2, kind, rows[i].Target)
		rows[i].SourceChanged = rows[i].NativeBefore != rows[i].NativeAfter
		rows[i].RuleAfter = "valid"
		if rows[i].Provider != "source" && rows[i].SourceChanged {
			rows[i].RuleAfter = "expired"
		}
	}
	c.Host.Accept(c.WS2, c.Rel1, "synthetic-host-decision-r1-with-language")
	for i := range rows {
		rows[i].Before = before.Evaluate(ctx, rows[i])
	}
	c.Host.Accept(c.WS2, c.Rel2, "synthetic-host-decision-r2")
	for i := range rows {
		rows[i].After = pre.Evaluate(ctx, rows[i])
	}
	// P-MIX-03: the old bundle shown right after the revision.
	mixed := pre.Admit(ctx, c.B1.Digest, c.B1)
	_, regenErr := pre.BuildBundle(ctx, []string{"en-US", "ja-JP"})
	mixedShown := pre.Show(ctx, "en-US")
	oldEN := map[string]bool{}
	for _, t := range c.S2.Conf.Current {
		if t.Locale == "en-US" && (t.Target == "order.batch_confirmed" || t.Target == "data:sales.order" || t.Target == "capability:sales.order.submit") {
			oldEN[t.Text] = true
		}
	}
	var leaked []string
	for _, sh := range mixedShown {
		if oldEN[sh.Text] {
			leaked = append(leaked, sh.Target)
		}
	}
	c.Rec.fault("pmix03-old-bundle-after-revision", "P-MIX-03", "4",
		!mixed.Admitted && mixed.LockCheck == "ok" && mixed.Integrity == "ok" && len(mixed.StaleSources) == 4 &&
			refused(regenErr, "stale-translation") && len(leaked) == 0,
		map[string]any{"admission": mixed, "regenerateBeforeReconfirm": errText(regenErr), "shown": mixedShown, "leakedOldTranslations": leaked})

	conf2, err := u.Confirm(c.S2.Conf, c.Rel2, c.Trans)
	if err != nil {
		return err
	}
	c.S4 = Scene{U: u, WS: c.WS2, Rel: c.Rel2, Conf: conf2, Grants: u.GrantAll(conf2), Host: c.Host, Providers: Languages}
	b2, err := c.S4.BuildBundle(ctx, []string{"en-US", "ja-JP"})
	if err != nil {
		return fmt.Errorf("stage 4 regenerate: %w", err)
	}
	c.B2 = b2
	reconfirmed := map[string]Cell{}
	for i := range rows {
		reconfirmed[rows[i].Kind+" "+rows[i].Target+" "+rows[i].Locale+" "+rows[i].Provider] = c.S4.Evaluate(ctx, rows[i])
	}
	for i := range rows {
		r := &rows[i]
		r.Agrees = r.Before.Status == "valid" && r.After.Status == r.Expected && r.Expected == r.RuleAfter
	}
	c.Matrix = rows
	oldPinned := c.S4.Admit(ctx, c.B1.Digest, c.B1)
	newPinned := c.S4.Admit(ctx, b2.Digest, b2)
	var expired, kept []map[string]string
	for _, r := range rows {
		item := map[string]string{"kind": r.Kind, "target": r.Target, "locale": r.Locale, "provider": r.Provider, "category": r.Category}
		if r.After.Status == "expired" {
			item["oldBasedOn"], item["reason"] = r.After.BasedOn, r.After.Diagnostic
			expired = append(expired, item)
		} else {
			kept = append(kept, item)
		}
	}
	en := map[string]string{}
	for _, m := range c.Rel2.Messages {
		if out, err := c.S4.ResolveMessage(ctx, m.ID, "en-US"); err == nil {
			en["message "+m.ID] = out.Resource.Text + " @" + out.Resource.SourceRevision
		}
	}
	for _, d := range b2.Displays {
		if d.Locale == "en-US" {
			en["display "+d.Resource.Target.Subject.ID] = d.Text + " @" + d.SourceRevision
		}
	}
	history := 0
	for _, h := range conf2.History {
		if h.Release == "r1" && h.Locale == "en-US" {
			history++
		}
	}
	c.Rec.AddStage(StageRecord{Stage: 4, Name: "source revision, dependent expiry, regeneration", LockFP: c.WS2.LockFP,
		Release: c.Rel2.Release, ReleaseHash: c.Rel2.Digest,
		Assets: []AssetCall{{"reference/message-localization", "ResolveWorkspace (per row)"},
			{"reference/multilingual-overlays", "BuildLockedDisplay (per row), BuildLockedLocaleBundle, CheckLocaleBundleAgainstLock"},
			{"reference/standard-messages", "Revision"}},
		Outputs: map[string]any{"bundleBefore": c.B1.Digest, "bundleAfter": b2.Digest, "releaseBefore": c.Rel1.Digest,
			"releaseAfter": c.Rel2.Digest, "matrix": rows, "expired": expired, "kept": kept, "afterReconfirm": reconfirmed,
			"oldPinnedAdmission": oldPinned, "newPinnedAdmission": newPinned, "englishAfterRegeneration": en,
			"historyKeptR1English": history}})
	return nil
}

// assetIdentities records the identifiers each asset used on its own and
// checks that mixing them into the unified record is refused.
func (c *Chain) IdentityChecks(ctx context.Context) error {
	u := c.U
	_, env, _, _, surfaces, _, _, _, err := overlays.ReferenceOrderCase()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(c.Repo, "reference", "multilingual-resources", "testdata", "order-entry.json"))
	if err != nil {
		return err
	}
	var fixture struct {
		Resources []struct {
			Key struct {
				Owner  string       `json:"owner"`
				Target modelref.Key `json:"target"`
				Field  string       `json:"field"`
			} `json:"key"`
			Source struct {
				Locale   string `json:"locale"`
				Revision string `json:"revision"`
			} `json:"source"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil || len(fixture.Resources) == 0 {
		return fmt.Errorf("resources fixture: %v", err)
	}
	fr := fixture.Resources[0]
	ov := surfaces[0]
	owner := string(u.Owner.ID) + "@" + u.Owner.Version
	c.Rec.Identities = []AssetIdentity{
		{"owner plugin", "reference/multilingual-overlays ReferenceOrderCase", string(ov.Target.ID) + "@" + ov.Target.Version, owner, "observed-by-call"},
		{"environment", "reference/multilingual-overlays ReferenceOrderCase", string(env.ID), string(u.Environment), "observed-by-call"},
		{"target", "reference/multilingual-overlays ReferenceOrderCase", ov.Subject.Scope + "/" + ov.Subject.Subject.Kind + "/" + ov.Subject.Subject.ID,
			u.Scope + "/semantic-capability/sales.order.submit", "observed-by-call"},
		{"owner plugin", "reference/multilingual-resources testdata/order-entry.json", fr.Key.Owner, owner, "read-from-file"},
		{"field", "reference/multilingual-resources testdata/order-entry.json", fr.Key.Field, u.DisplayField, "read-from-file"},
		{"source revision", "reference/multilingual-resources testdata/order-entry.json", fr.Source.Revision, "native rule per asset + release digest", "read-from-file"},
		{"owner plugin", "reference/message-localization adapter_test.go", "sales.order_ui@1.0.0", owner, "read-from-test-source"},
		{"source locale", "reference/message-localization adapter_test.go", "en-US", u.SourceLocale, "read-from-test-source"},
		{"field", "reference/message-localization adapter.go", "message-body", u.MessageField, "read-from-test-source"},
		{"tenant/record", "reference/multilingual-business-data postgres_test.go", "store-a / product:P-100", u.Tenant + " / product:<itemCode>", "read-from-test-source"},
		{"owner plugin", "reference/multilingual-overlays workspace_display_test.go", "tei.order.ui@1.0.0", owner, "read-from-test-source"},
	}
	s := c.S2
	type mix struct{ id, want string; err error }
	var mixes []mix
	foreignOwner := ov.Target
	// message owner of another asset
	req := s.messageRequest(ctx, "order.batch_confirmed", localization.Policy{Locale: "en-US"}, u.MessageRecords(s.Conf, nil))
	req.Request.Message.Owner = foreignOwner
	_, err = localization.ResolveWorkspace(req)
	mixes = append(mixes, mix{"message-owner-of-overlays-case", "message-owner-unselected", err})
	// environment of another asset
	req = s.messageRequest(ctx, "order.batch_confirmed", localization.Policy{Locale: "en-US"}, u.MessageRecords(s.Conf, nil))
	req.Environment = env.ID
	_, err = localization.ResolveWorkspace(req)
	mixes = append(mixes, mix{"environment-of-overlays-case", "missing-environment", err})
	// a translation record in the source language (the fixture direction was reversed)
	rec := u.MessageRecords(s.Conf, nil)
	for i := range rec {
		if rec[i].MessageID == "order.batch_confirmed" && rec[i].Locale == "en-US" {
			rec[i].Locale = u.SourceLocale
		}
	}
	grants := append([]localization.Grant(nil), s.Grants.Messages...)
	for _, g := range s.Grants.Messages {
		if g.MessageID == "order.batch_confirmed" {
			g.Locale = u.SourceLocale
			grants = append(grants, g)
		}
	}
	req = s.messageRequest(ctx, "order.batch_confirmed", localization.Policy{Locale: "en-US"}, rec)
	req.Grants = grants
	_, err = localization.ResolveWorkspace(req)
	mixes = append(mixes, mix{"translation-in-source-locale", "source-override-unsupported", err})
	// grant of another environment
	req = s.messageRequest(ctx, "order.batch_confirmed", localization.Policy{Locale: "en-US"}, u.MessageRecords(s.Conf, nil))
	req.Grants = nil
	for _, g := range s.Grants.Messages {
		g.Environment = "production"
		req.Grants = append(req.Grants, g)
	}
	_, err = localization.ResolveWorkspace(req)
	mixes = append(mixes, mix{"grant-of-other-environment", "translation-not-granted", err})
	// overlay bundle for the owner of another asset
	graph, _ := u.Graph(s.WS, s.Rel)
	_, err = overlays.BuildLockedLocaleBundle(s.WS.Catalog, s.WS.Env, s.WS.Lock, graph, u.Surfaces(s.Rel), u.Packs(s.Conf, nil),
		s.Grants.Displays, nil, overlays.LocaleBundleRequest{Target: foreignOwner, Keys: u.DisplayKeys(s.Rel), Locales: []string{"en-US"}})
	mixes = append(mixes, mix{"bundle-target-of-overlays-case", "unselected-target", err})
	// pack pinned to another owner version
	packs := u.Packs(s.Conf, nil)
	packs[0].Target = plugin.Identity{ID: u.Owner.ID, Version: "1.2.0"}
	_, err = overlays.BuildLockedLocaleBundle(s.WS.Catalog, s.WS.Env, s.WS.Lock, graph, u.Surfaces(s.Rel), packs,
		s.Grants.Displays, nil, overlays.LocaleBundleRequest{Target: u.Owner, Keys: u.DisplayKeys(s.Rel), Locales: []string{"en-US"}})
	mixes = append(mixes, mix{"pack-for-other-owner-version", "incompatible-plugin-version", err})
	// the resources fixture's field name
	sf := u.Surfaces(s.Rel)
	sf[0].Field = fr.Key.Field
	_, err = overlays.BuildLockedLocaleBundle(s.WS.Catalog, s.WS.Env, s.WS.Lock, graph, sf, nil, nil, nil,
		overlays.LocaleBundleRequest{Target: u.Owner, Keys: u.DisplayKeys(s.Rel), Locales: []string{"ja-JP"}})
	mixes = append(mixes, mix{"field-of-resources-fixture", "unsupported-display-field", err})
	// the overlays/resources target form (scope sales, kind function)
	foreignKey := modelref.Key{Scope: fr.Key.Target.Scope, Subject: subject.Subject{Kind: fr.Key.Target.Subject.Kind, ID: fr.Key.Target.Subject.ID}}
	present := s.WS.HasNode(foreignKey)
	if present {
		err = nil
	} else {
		err = fmt.Errorf("target %s/%s/%s is not in the Derived Index", foreignKey.Scope, foreignKey.Subject.Kind, foreignKey.Subject.ID)
	}
	mixes = append(mixes, mix{"target-form-of-resources-fixture", "not in the Derived Index", err})
	all := true
	var observed []map[string]string
	for _, m := range mixes {
		ok := refused(m.err, m.want)
		all = all && ok
		observed = append(observed, map[string]string{"case": m.id, "want": m.want, "got": errText(m.err)})
	}
	c.Rec.Completion = append(c.Rec.Completion, Check{ID: "identity-unified-and-mixing-refused", Stage: "1-4",
		Passed: all, Observed: observed})
	return nil
}
