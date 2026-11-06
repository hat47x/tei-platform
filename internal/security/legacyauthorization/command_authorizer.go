package legacyauthorization

import (
	"context"
	"errors"
	"fmt"

	"tei/internal/command"
)

// CommandAuthorizer connects an existing authorization source to the Command
// boundary. It answers only "may this subject run this capability on this
// resource"; the Command boundary and the Application still decide whether the
// capability is declared and reachable.
//
// The mapping is explicit because a capability is not an action and an
// Application is not a resource by definition:
//
//   - Action maps a capability to the action the existing rules evaluate.
//   - Resource maps a Command request to the resource those rules evaluate.
//
// A menu or a Pattern is never consulted. When the subject lacks the grant,
// the call is denied even if the Pattern advertises the command, which is the
// case AC-11 asks to demonstrate.
type CommandAuthorizer struct {
	// Subject is the authenticated subject identity. A Command request carries
	// inputs, not an identity, so the host supplies it here.
	Subject string
	// Action maps a capability to an action. A nil mapper uses the capability ID.
	Action func(command.Request) string
	// Resource maps a request to a resource. A nil mapper uses the application ID.
	Resource func(command.Request) string

	Store       SubjectStore
	Rule        Rule
	EpochReader EpochReader
}

// Authorize denies the call unless the existing rules allow the subject to run
// the capability on the resource.
func (a CommandAuthorizer) Authorize(ctx context.Context, request command.Request) error {
	if a.Subject == "" {
		return fmt.Errorf("command authorization has no subject identity")
	}
	action := string(request.Capability)
	if a.Action != nil {
		action = a.Action(request)
	}
	resource := string(request.Application)
	if a.Resource != nil {
		resource = a.Resource(request)
	}
	if action == "" || resource == "" {
		return fmt.Errorf("command authorization has no action or resource for %s", request.Capability)
	}
	adapter := Adapter{Store: a.Store, Rule: a.Rule, EpochReader: a.EpochReader}
	result, err := adapter.Invoke(ctx, AuthorizationCapability, Input{
		SubjectID: a.Subject,
		Action:    action,
		Resource:  resource,
	})
	if err != nil {
		return err
	}
	if result.Outcome != "allowed" {
		reason := "denied"
		if denial, ok := result.Payload.(AuthorizationDenial); ok && denial.Reason != "" {
			reason = denial.Reason
		}
		return fmt.Errorf("subject is not authorized for %s on %s: %s: %w", action, resource, reason, ErrNoGrant)
	}
	return nil
}

// ErrNoGrant reports a denial that a rule produced on purpose, so a caller can
// distinguish "the rules answered no" from "the rules could not be evaluated".
var ErrNoGrant = errors.New("not authorized")
