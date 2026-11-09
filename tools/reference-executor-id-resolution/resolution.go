// Package executoridresolution is a read-only reference for C-3 of the
// composition connect-point decision: instead of naming a deployment executor's
// CapabilityID in code, resolve it from the Environment's binding and check that
// the plugin lock actually admits it.
//
// It answers three questions with existing functions only:
//  1. which executor does the binding select for this action?  (routing.RouteCommand)
//  2. does the resolved ID exist as a deployment-executor capability in the
//     plan built from the plugin lock?               (plugin.BuildPlan + Capability)
//  3. does that capability support the action?       (Capability.SupportsAction)
//
// It reads the Workspace and the lock; it writes nothing and grants nothing.
package executoridresolution

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/deployment/routing"
	"tei/internal/environment"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/workspace"
)

const (
	// Schema identifies the report. It is a reference output, not a product
	// contract.
	Schema = "tei.reference.executor-id-resolution/v0"
	// Authority states what the report is allowed to claim.
	Authority = "derived-reference"

	RuleID      = "tei.reference.executor-id-resolution"
	RuleVersion = "v0"
)

// rulePolicy is covered by the fingerprint.
const rulePolicy = `executor-id-resolution/v0
resolution: routing.RouteCommand over the Environment's deployment-executor binding
admission: pruntime.BuildPlan over the World's plugins/bundles and the supplied
           plugin lock, then EnvironmentPlan.Capability(deployment-executor, id)
support: plugin.Capability.SupportsAction for the requested action kind
refused when: the binding is missing or ambiguous, the resolved id is absent from
              the plan, or the capability does not support the action
not verified: that the executor exists, that its artifact is trusted, or that an
              acknowledgement reflects external state
not issued: authentication, authority, permission, or a durable receipt
`

// Status codes name the outcome of the resolution.
const (
	StatusResolved       = "RESOLVED"
	StatusBindingMissing = "BINDING_MISSING"
	StatusNotInPlan      = "NOT_IN_PLAN"
	StatusUnsupported    = "ACTION_UNSUPPORTED"
	StatusLockRejected   = "LOCK_REJECTED"
)

// Report is the whole answer.
type Report struct {
	Schema          string `json:"schema"`
	Authority       string `json:"authority"`
	RuleID          string `json:"ruleId"`
	RuleVersion     string `json:"ruleVersion"`
	RuleFingerprint string `json:"ruleFingerprint"`
	Environment     string `json:"environment"`
	Action          string `json:"action"`
	// ResolvedID is the executor the binding selects. It is empty when the
	// binding is missing or ambiguous.
	ResolvedID string `json:"resolvedId,omitempty"`
	Status     string `json:"status"`
	// Admitted reports whether the plan contains that deployment-executor
	// capability. It is present even when the capability does not support the
	// action, so a reader can tell "absent" from "present but unsupported".
	Admitted bool `json:"admitted"`
	// HardCodedID is the id the reference implementation currently embeds, when
	// the caller supplies one. Reporting it next to ResolvedID is the point of
	// this reference: the two must agree before the hard-coded value is dropped.
	HardCodedID string   `json:"hardCodedId,omitempty"`
	Detail      string   `json:"detail,omitempty"`
	Notes       []string `json:"notes"`
}

// Input is what the caller supplies. Lock is the decoded plugin lock file; the
// reference never writes one.
type Input struct {
	WorkspaceRoot string
	Environment   environment.Environment
	Action        string
	// Lock is the plugin lock for this environment.
	Lock plugin.Lock
	// HardCodedID is the id a reference implementation currently names in code.
	// It is compared with the resolved id, not used for resolution.
	HardCodedID string
	// Database and Application scope the command so a target-scoped binding can
	// match. Empty values keep the command global.
	Database    core.CanonicalID
	Application core.CanonicalID
}

// Resolve answers the three questions. It never invents an executor id.
func Resolve(in Input) (Report, error) {
	loaded, err := workspace.Load(in.WorkspaceRoot)
	if err != nil {
		return Report{}, fmt.Errorf("load Workspace: %w", err)
	}
	report := Report{
		Schema: Schema, Authority: Authority,
		RuleID: RuleID, RuleVersion: RuleVersion,
		RuleFingerprint: fingerprint(),
		Environment:     string(in.Environment.ID),
		Action:          in.Action,
		HardCodedID:     in.HardCodedID,
		Notes:           notes(),
	}

	// 1. The binding selects the executor. The command only needs the action
	// to match a deployment-executor binding.
	cmd := executor.Command{
		ActionID: "reference/" + in.Action, Kind: in.Action,
		Database: in.Database, Application: in.Application,
	}
	routed, routeErr := routing.RouteCommand(in.Environment, cmd)
	if routeErr != nil {
		report.Status = StatusBindingMissing
		report.Detail = routeErr.Error()
		return report, nil
	}
	report.ResolvedID = string(routed.ExecutorID)

	// 2. The plan built from the lock decides whether that id is admitted.
	plan, planErr := pruntime.BuildPlan(
		plugin.Catalog{Plugins: loaded.World.Plugins, Bundles: loaded.World.Bundles},
		in.Environment, in.Lock,
	)
	if planErr != nil {
		report.Status = StatusLockRejected
		report.Detail = planErr.Error()
		return report, nil
	}
	capability, ok := plan.Capability(plugin.CapabilityDeploymentExecutor, routed.ExecutorID)
	if !ok {
		report.Status = StatusNotInPlan
		report.Detail = "the plan has no deployment-executor capability with this id, or it is ambiguous"
		return report, nil
	}
	report.Admitted = true

	// 3. The capability itself decides whether it supports the action.
	if !capability.Capability.SupportsAction(deployment.ActionKind(in.Action)) {
		report.Status = StatusUnsupported
		report.Detail = "the admitted capability does not support this action"
		return report, nil
	}
	report.Status = StatusResolved
	return report, nil
}

func notes() []string {
	return []string{
		"resolution uses the Environment binding, not a name written in code",
		"admission means the plugin lock produced a plan containing that capability",
		"neither admission nor support proves the executor works or that an effect occurred",
		"hardCodedId is reported for comparison only; it does not take part in resolution",
		"no authentication, authority, permission or durable receipt is issued here",
	}
}

func fingerprint() string { return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(rulePolicy))) }

// Marshal renders the report as stable, indented JSON.
func (r Report) Marshal() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
