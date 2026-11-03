package effectcommit

import (
	"errors"
	"fmt"
	"reflect"
)

// GuaranteeLevel is an unapproved A2 reference proposal, not a ratified
// contract. It lets a Provider DECLARE what atomicity it claims (Issue #1174
// requirement 4) so that a Target composition can refuse a Provider that
// cannot couple authorization and effect. A declaration is a claim by the
// Provider author; it is not proof, and it never replaces a Provider
// conformance test against the real backing store.
type GuaranteeLevel string

const (
	// GuaranteeUndeclared is what a Provider that implements no declaration
	// is reported as. It is never accepted by RequireGuarantee (fail closed),
	// but existing code that does not call RequireGuarantee is unaffected.
	GuaranteeUndeclared GuaranteeLevel = "undeclared"
	// GuaranteeNotAtomic: authorization check and effect cannot be made one
	// atomic unit (for example a remote SaaS or an event queue).
	GuaranteeNotAtomic GuaranteeLevel = "not-atomic"
	// GuaranteeAtomicWithCommit: the authority check, the effect and the
	// receipt are claimed to take part in one commit.
	GuaranteeAtomicWithCommit GuaranteeLevel = "atomic-with-commit"
)

// Guarantee is a Provider's declared atomicity level with a human-readable
// reason. Reason is for operators; it must not carry secrets.
type Guarantee struct {
	Level  GuaranteeLevel
	Reason string
}

// GuaranteeDeclarer is OPTIONAL. A Provider that does not implement it keeps
// working exactly as before.
//
// A wrapper type that EMBEDS a concrete declaring Provider has that Guarantee
// method promoted, so it would inherit the inner level even if the wrapper adds
// non-atomic behaviour around it. A wrapper must therefore declare its OWN
// level explicitly. (Embedding the Provider INTERFACE hides the declaration,
// which is the fail-closed direction.)
type GuaranteeDeclarer interface {
	Guarantee() Guarantee
}

var (
	// ErrGuaranteeNotMet means the Provider declared a level other than the
	// required one.
	ErrGuaranteeNotMet = errors.New("確定境界の保証水準が要求と一致しません")
	// ErrGuaranteeUndeclared means the Provider declared nothing (or an
	// unknown level); fail closed.
	ErrGuaranteeUndeclared = errors.New("確定境界の保証水準が宣言されていません")
)

func knownLevel(l GuaranteeLevel) bool {
	return l == GuaranteeNotAtomic || l == GuaranteeAtomicWithCommit
}

// isNilProvider reports a nil interface AND a typed-nil pointer (a nil
// *T stored in the interface), which would otherwise pass a type assertion and
// could even be "declared" by a pointer-receiver method that never dereferences
// its receiver.
func isNilProvider(p Provider) bool {
	if p == nil {
		return true
	}
	switch v := reflect.ValueOf(p); v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Func, reflect.Interface, reflect.Chan:
		return v.IsNil()
	}
	return false
}

// DeclaredGuarantee returns the Provider's declaration, normalised: a nil
// (or typed-nil) Provider, a Provider without GuaranteeDeclarer, an unknown
// level, or a Guarantee() that panics yields GuaranteeUndeclared. A panic is
// treated as "undeclared" (fail closed) rather than allowed to escape: the
// Runner reads the declaration AFTER an effect may have committed, so a panic
// here must never change what the caller sees.
func DeclaredGuarantee(p Provider) (g Guarantee) {
	undeclared := Guarantee{Level: GuaranteeUndeclared}
	if isNilProvider(p) {
		return undeclared
	}
	d, ok := p.(GuaranteeDeclarer)
	if !ok {
		return undeclared
	}
	defer func() {
		if recover() != nil {
			g = undeclared
		}
	}()
	g = d.Guarantee()
	if !knownLevel(g.Level) {
		return undeclared
	}
	return g
}

// RequireGuarantee fails closed unless the Provider declared exactly the
// wanted level. want must be GuaranteeAtomicWithCommit or GuaranteeNotAtomic.
func RequireGuarantee(p Provider, want GuaranteeLevel) error {
	if isNilProvider(p) {
		return ErrMissingProvider
	}
	if !knownLevel(want) {
		return ErrInvalidRequest
	}
	got := DeclaredGuarantee(p)
	if got.Level == GuaranteeUndeclared {
		return ErrGuaranteeUndeclared
	}
	if got.Level != want {
		return fmt.Errorf("%w: 宣言=%s 要求=%s", ErrGuaranteeNotMet, got.Level, want)
	}
	return nil
}

// NewRequiredRunner is the composition helper a Target uses: it returns a
// Runner only if the Provider satisfies the required level, so a refused
// Provider never reaches Execute through this path. A Target that builds
// Runner{} by hand bypasses this check; that is a composition-review matter,
// not something this package can enforce.
func NewRequiredRunner(p Provider, want GuaranteeLevel, audit AuditConfig) (Runner, error) {
	if err := RequireGuarantee(p, want); err != nil {
		return Runner{}, err
	}
	return Runner{Provider: p, Audit: audit}, nil
}
