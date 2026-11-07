package hook

import (
	"context"

	"tei/internal/core"
)

// Guardedは、規則の判定が通ったときだけ、宣言した能力を呼ぶImplementationである。
// 判定が通らなければNotSatisfiedを返し、能力は呼ばない。呼出しはRunnerが渡す
// Caller（宣言で制限済み）を通るので、宣言外の能力へは届かない。
//
// 規則（ReferenceRule）と能力の組を設定で与えるため、同じ宣言と同じ入力からは
// 同じ結果になる。実Pluginはこの契約を実装し、ここは参照実装である。
type Guarded struct {
	Rule ReferenceRule
	// Capabilityは、判定が通ったときに呼ぶ能力である。フックの宣言の`uses`に
	// 載っている必要がある。載っていなければDeclaredが拒否し、境界へは届かない。
	Capability core.CanonicalID
	// Inputsは、能力へ渡す入力である。
	Inputs map[string]string
}

// Runは、呼出し元を持たない経路のための実装である。規則は判定するが能力を
// 呼べないため、判定が通った場合は失敗として返す（呼ばずに成功にしない）。
func (g Guarded) Run(_ context.Context, invocation Invocation) error {
	if err := g.Rule.evaluate(invocation); err != nil {
		return err
	}
	return &Error{
		Code:    "HOOK_CALLER_UNAVAILABLE",
		Message: "the guarded hook needs a caller to run capability " + string(g.Capability),
	}
}

// RunWithCallerは、規則を判定し、通ったときだけ能力を呼ぶ。
func (g Guarded) RunWithCaller(ctx context.Context, invocation Invocation, caller Caller) error {
	if err := g.Rule.evaluate(invocation); err != nil {
		return err
	}
	if g.Capability == "" {
		return &Error{Code: "HOOK_CAPABILITY_NOT_CONFIGURED", Message: "the guarded hook has no capability to call"}
	}
	if caller == nil {
		return &Error{Code: "HOOK_IMPLEMENTATION_UNAVAILABLE", Message: "no caller is configured for this hook boundary"}
	}
	_, err := caller.Call(ctx, g.Capability, g.Inputs)
	return err
}

// Guardedは、呼出し元を受け取る実装として使える。
var _ CallingImplementation = Guarded{}
