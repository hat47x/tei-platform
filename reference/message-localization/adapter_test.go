package messagelocalization

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
	messages "tei/reference/standard-messages"
)

var (
	msgPlugin = plugin.Identity{ID: "std.message", Version: "1.0.0"}
	owner     = plugin.Identity{ID: "sales.order_ui", Version: "1.0.0"}
	jp        = plugin.Identity{ID: "std.localization_ja", Version: "1.0.0"}
	jpOther   = plugin.Identity{ID: "custom.localization_ja", Version: "1.0.0"}
)

func fixture(t *testing.T, localized bool) Input {
	t.Helper()
	def := messages.Definition{
		Ref: messages.Ref{Owner: owner, ID: core.CanonicalID("order.confirmed")},
		SourceLocale: "en-US", Text: "Order {orderId} confirmed", Format: messages.FormatNamedV0,
		Parameters: []messages.Parameter{{Name: "orderId", Kind: messages.ParameterText}},
	}
	defs := []plugin.Definition{
		{ID: msgPlugin.ID, Version: msgPlugin.Version, Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
		{ID: owner.ID, Version: owner.Version, Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Requires: []plugin.Requirement{{Plugin: msgPlugin.ID, CompatibleMajor: 1}}},
	}
	env := environment.Environment{ID: core.CanonicalID("development"), Plugins: []environment.ExtensionRequirement{{ID: owner.ID, CompatibleMajor: 1}}}
	if localized {
		defs = append(defs, plugin.Definition{ID: jp.ID, Version: jp.Version, Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
			Requires: []plugin.Requirement{{Plugin: msgPlugin.ID, CompatibleMajor: 1}}})
		env.Plugins = append(env.Plugins, environment.ExtensionRequirement{ID: jp.ID, CompatibleMajor: 1})
	}
	catalog := plugin.Catalog{Plugins: defs}
	lock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	input := Input{Plugins: catalog, Environment: env, Lock: lock, Messages: messages.Catalog{Definitions: []messages.Definition{def}},
		Policy: Policy{Locale: "en-US"}, Request: Request{Application: "order_entry", Message: def.Ref},
		Review: func(ReviewInput) error { return nil },
	}
	if localized {
		input.Stores = []StoreDeclaration{{Provider: jp, ID: "translations", Schema: StoreSchema}}
		input.Bindings = []ApplicationBinding{{Provider: jp, StoreID: "translations", Application: "order_entry", Environment: "development"}}
		input.Records = []TranslationRecord{{Application: "order_entry", StoreID: "translations", Provider: jp, Target: owner,
			MessageID: def.Ref.ID, Locale: "ja-JP", Text: "注文 {orderId} を確定しました", BasedOn: messages.Revision(def), Review: "confirmed"}}
		input.Grants = []Grant{{Environment: "development", Application: "order_entry", Provider: jp, Target: owner,
			MessageID: def.Ref.ID, Locale: "ja-JP"}}
		input.Policy.Locale = "ja-JP"
	}
	return input
}

func TestStandardMessageWithoutLocalizationPlugin(t *testing.T) {
	in := fixture(t, false)
	out, err := Resolve(in)
	if err != nil || out.Resource.Text != "Order {orderId} confirmed" || !out.Resource.FromSource || out.Resource.Locale != "en-US" {
		t.Fatalf("base message unavailable: %+v %v", out, err)
	}
	if out.LockFingerprint == "" {
		t.Fatal("lost source plugin lock")
	}
}

func TestLocalizationExtendsStandardMessageWithoutMutatingIt(t *testing.T) {
	in := fixture(t, true)
	original := in.Messages.Definitions[0]
	called := false
	in.Review = func(review ReviewInput) error {
		called = true
		if review.Request != in.Request || review.LockFingerprint == "" || review.SourceRevision != messages.Revision(original) ||
			len(review.Bindings) != 1 || len(review.Records) != 1 || len(review.Grants) != 1 {
			return errors.New("review witness mismatch")
		}
		return nil
	}
	out, err := Resolve(in)
	if err != nil || !called || out.Resource.Text != "注文 {orderId} を確定しました" || out.Resource.FromSource ||
		out.Resource.SourceRevision != messages.Revision(original) {
		t.Fatalf("localized display: %+v %v", out, err)
	}
	if in.Messages.Definitions[0].Text != original.Text {
		t.Fatal("ordinary message mutated")
	}
}

func TestHostMustApproveBeforeDisplay(t *testing.T) {
	for _, localized := range []bool{false, true} {
		in := fixture(t, localized)
		in.Review = nil
		if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), "host-review-required") {
			t.Fatalf("nil review accepted: %v", err)
		}
		in.Review = func(ReviewInput) error { return errors.New("revoked") }
		if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), "host-review-denied") {
			t.Fatalf("revoked review accepted: %v", err)
		}
	}
}

func TestApplicationBindingAndGrantCannotBeSelfDeclared(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Input)
		code   string
	}{
		{"not-applied", func(i *Input) { i.Bindings = nil }, "unapplied-plugin-store"},
		{"foreign-environment", func(i *Input) { i.Bindings[0].Environment = "production" }, "unapplied-plugin-store"},
		{"foreign-application", func(i *Input) { i.Bindings[0].Application = "other_app" }, "unapplied-plugin-store"},
		{"different-schema", func(i *Input) { i.Stores[0].Schema = "custom.schema" }, "unapplied-plugin-store"},
		{"no-grant", func(i *Input) { i.Grants = nil }, "translation-not-granted"},
		{"foreign-grant", func(i *Input) { i.Grants[0].Application = "other_app" }, "translation-not-granted"},
		{"unselected-provider", func(i *Input) { i.Records[0].Provider = jpOther }, "unselected-translation-provider"},
		{"source-language-replacement", func(i *Input) { i.Records[0].Locale = "en-US"; i.Grants[0].Locale = "en-US" }, "source-override-unsupported"},
		{"missing-placeholder", func(i *Input) { i.Records[0].Text = "注文を確定しました" }, "invalid-translation-contract"},
		{"unrecognized-format", func(i *Input) { i.Messages.Definitions[0].Format = "mf2" }, "message-catalog"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			in := fixture(t, true)
			tt.change(&in)
			if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), tt.code) {
				t.Fatalf("expected %s, got %v", tt.code, err)
			}
		})
	}
}

func TestTranslationRefreshAndUnconfirmedAreRejected(t *testing.T) {
	for _, review := range []string{"candidate", "needs-review"} {
		in := fixture(t, true)
		in.Records[0].Review = review
		if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), "unapproved-translation") {
			t.Fatalf("%s was used: %v", review, err)
		}
	}
	in := fixture(t, true)
	in.Messages.Definitions[0].Parameters[0].Kind = messages.ParameterInteger
	if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), "stale-translation") {
		t.Fatalf("old parameter contract accepted: %v", err)
	}
	in = fixture(t, true)
	in.Messages.Definitions[0].Text = "Confirmed order {orderId}"
	if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), "stale-translation") {
		t.Fatalf("old source accepted: %v", err)
	}
}

func TestExplicitSourceOnlyPolicySuppressesTranslation(t *testing.T) {
	in := fixture(t, true)
	in.Policy = Policy{Mode: "source-only", Locale: "ja-JP", AllowSourceFallback: true}
	out, err := Resolve(in)
	if err != nil || out.Resource.Text != "Order {orderId} confirmed" || !out.Resource.FromSource || !out.Resource.Fallback {
		t.Fatalf("source-only policy ignored: %+v %v", out, err)
	}
	in.Policy.AllowSourceFallback = false
	if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), "missing-translation") {
		t.Fatalf("unsafe fallback: %v", err)
	}
}

func TestNoImplicitFallbackAndCriticalMessage(t *testing.T) {
	in := fixture(t, true)
	in.Policy.Locale = "fr-FR"
	if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), "missing-translation") {
		t.Fatalf("unexpected fallback: %v", err)
	}
	in.Policy.Fallback = []string{"ja-JP"}
	out, err := Resolve(in)
	if err != nil || out.Resource.Locale != "ja-JP" || !out.Resource.Fallback {
		t.Fatalf("explicit fallback failed: %+v %v", out, err)
	}
	in.Messages.Definitions[0].Critical = true
	in.Records[0].BasedOn = messages.Revision(in.Messages.Definitions[0])
	if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), "missing-translation") {
		t.Fatalf("critical fallback accepted: %v", err)
	}
}

func TestCollisionRequiresExplicitFullReplacement(t *testing.T) {
	in := fixture(t, true)
	in.Plugins.Plugins = append(in.Plugins.Plugins, plugin.Definition{
		ID: jpOther.ID, Version: jpOther.Version, Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Requires: []plugin.Requirement{{Plugin: msgPlugin.ID, CompatibleMajor: 1}},
	})
	in.Environment.Plugins = append(in.Environment.Plugins, environment.ExtensionRequirement{ID: jpOther.ID, CompatibleMajor: 1})
	var err error
	in.Lock, err = plugin.BuildLock(in.Plugins, []environment.Environment{in.Environment})
	if err != nil {
		t.Fatal(err)
	}
	second := in.Records[0]
	second.Provider = jpOther
	second.Text = "注文 {orderId} の受付完了"
	second.StoreID = "other_translations"
	in.Records = append(in.Records, second)
	in.Stores = append(in.Stores, StoreDeclaration{Provider: jpOther, ID: second.StoreID, Schema: StoreSchema})
	in.Bindings = append(in.Bindings, ApplicationBinding{Environment: "development", Application: "order_entry", Provider: jpOther, StoreID: second.StoreID})
	in.Grants = append(in.Grants, Grant{Environment: "development", Application: "order_entry", Provider: jpOther, Target: owner, MessageID: second.MessageID, Locale: "ja-JP"})
	if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), "ambiguous-translation") {
		t.Fatalf("collision silently won: %v", err)
	}
	in.Overrides = []Override{{Application: "order_entry", Target: owner, MessageID: second.MessageID, Locale: "ja-JP", Winner: jpOther, Replaces: jp}}
	out, err := Resolve(in)
	if err != nil || out.Resource.Text != second.Text {
		t.Fatalf("explicit winner ignored: %+v %v", out, err)
	}
	in.Overrides = append(in.Overrides, Override{Application: "order_entry", Target: owner, MessageID: second.MessageID, Locale: "ja-JP", Winner: jp, Replaces: jpOther})
	if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), "ambiguous-translation") {
		t.Fatalf("contradictory override accepted: %v", err)
	}
}

func TestNoCrossApplicationRecordLeakage(t *testing.T) {
	in := fixture(t, true)
	in.Records[0].Application = "other_app"
	if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), "missing-translation") {
		t.Fatalf("other app translation leaked: %v", err)
	}
}

func TestDifferentPluginLockIsRejectedBeforeReview(t *testing.T) {
	in := fixture(t, true)
	in.Lock.Environments[0].Environment = "other_environment"
	called := false
	in.Review = func(ReviewInput) error { called = true; return nil }
	if _, err := Resolve(in); err == nil || !strings.Contains(err.Error(), "plugin-lock") || called {
		t.Fatalf("foreign lock accepted: called=%v err=%v", called, err)
	}
}
