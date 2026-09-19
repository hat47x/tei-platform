package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	actorContextR3AKLifecyclePrepared           = "prepared"
	actorContextR3AKLifecycleResponseCorrelated = "response-correlated"
	actorContextR3AKCorrelationMatchedCurrent   = "matched-current-revision"
)

type actorContextR3AKRequestRecordProbe struct {
	RequestIdentity         string
	RequestRevisionIdentity string
	LifecycleState          string
	Request                 actorContextR3AJAuthorityInformationRequestProbe
	AuthorityLimits         []string
}

type actorContextR3AKResponseWitnessProbe struct {
	RequestIdentity         string
	RequestRevisionIdentity string
	ResponseRef             string
	ProvidedInformationClasses []string
	EvidenceRefs            []string
}

type actorContextR3AKCorrelationResultProbe struct {
	RequestIdentity         string
	RequestRevisionIdentity string
	ResponseRef             string
	CorrelationStatus       string
	LifecycleState          string
	ProvidedInformationClasses []string
	EvidenceRefs            []string
	AuthorityLimits         []string
}

func TestActorContextR3AKBuildsStableRequestAndRevisionIdentity(t *testing.T) {
	request := actorContextR3AKCurrentRequest(t)
	record, err := actorContextR3AKPrepareRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if record.RequestIdentity == "" || record.RequestRevisionIdentity == "" {
		t.Fatalf("R3AK lost request identities: %+v", record)
	}
	if record.LifecycleState != actorContextR3AKLifecyclePrepared {
		t.Fatalf("R3AK initial lifecycle=%q want=%q", record.LifecycleState, actorContextR3AKLifecyclePrepared)
	}

	reordered := request
	reordered.MachineReasons = actorContextR3AKReverseStrings(request.MachineReasons)
	reordered.InformationNeedClasses = actorContextR3AKReverseStrings(request.InformationNeedClasses)
	reordered.EvidenceNeedClasses = actorContextR3AKReverseStrings(request.EvidenceNeedClasses)
	reordered.KnownSourceRefs = actorContextR3AKReverseStrings(request.KnownSourceRefs)
	reordered.DrillDownRefs = actorContextR3AKReverseStrings(request.DrillDownRefs)

	second, err := actorContextR3AKPrepareRequest(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if record.RequestIdentity != second.RequestIdentity ||
		record.RequestRevisionIdentity != second.RequestRevisionIdentity {
		t.Fatalf("R3AK input order changed identity: first=%+v second=%+v", record, second)
	}
}

func TestActorContextR3AKSourceLocatorChangeOnlyChangesRequestRevision(t *testing.T) {
	request := actorContextR3AKCurrentRequest(t)
	first, err := actorContextR3AKPrepareRequest(request)
	if err != nil {
		t.Fatal(err)
	}

	changed := request
	changed.KnownSourceRefs = append([]string(nil), request.KnownSourceRefs...)
	changed.KnownSourceRefs[0] = changed.KnownSourceRefs[0] + ".moved"
	second, err := actorContextR3AKPrepareRequest(changed)
	if err != nil {
		t.Fatal(err)
	}

	if first.RequestIdentity != second.RequestIdentity {
		t.Fatalf("R3AK locator-only context change altered semantic request identity: first=%q second=%q",
			first.RequestIdentity, second.RequestIdentity)
	}
	if first.RequestRevisionIdentity == second.RequestRevisionIdentity {
		t.Fatalf("R3AK locator-only context change did not alter request revision identity")
	}
}

func TestActorContextR3AKMatchingIdentityAndRevisionCorrelatesWithoutAcceptingAuthority(t *testing.T) {
	request := actorContextR3AKCurrentRequest(t)
	record, err := actorContextR3AKPrepareRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	response := actorContextR3AKResponseWitnessProbe{
		RequestIdentity:         record.RequestIdentity,
		RequestRevisionIdentity: record.RequestRevisionIdentity,
		ResponseRef:             "test-only:r3ak/response-1",
		ProvidedInformationClasses: []string{
			"authorization-evaluation-freshness-authority",
		},
		EvidenceRefs: []string{"test-only:r3ak/evidence-1"},
	}

	got, err := actorContextR3AKCorrelateResponse(record, response)
	if err != nil {
		t.Fatal(err)
	}
	if got.CorrelationStatus != actorContextR3AKCorrelationMatchedCurrent ||
		got.LifecycleState != actorContextR3AKLifecycleResponseCorrelated {
		t.Fatalf("R3AK lost correlation state: %+v", got)
	}
	if got.RequestIdentity != record.RequestIdentity ||
		got.RequestRevisionIdentity != record.RequestRevisionIdentity ||
		got.ResponseRef != response.ResponseRef {
		t.Fatalf("R3AK correlation changed identity: %+v", got)
	}
	if !reflect.DeepEqual(got.ProvidedInformationClasses, response.ProvidedInformationClasses) ||
		!reflect.DeepEqual(got.EvidenceRefs, response.EvidenceRefs) {
		t.Fatalf("R3AK correlation lost response witness detail: %+v", got)
	}
}

func TestActorContextR3AKStaleRequestRevisionFailsClosed(t *testing.T) {
	request := actorContextR3AKCurrentRequest(t)
	current, err := actorContextR3AKPrepareRequest(request)
	if err != nil {
		t.Fatal(err)
	}

	staleRequest := request
	staleRequest.DrillDownRefs = append([]string(nil), request.DrillDownRefs...)
	staleRequest.DrillDownRefs[0] = staleRequest.DrillDownRefs[0] + ".old"
	stale, err := actorContextR3AKPrepareRequest(staleRequest)
	if err != nil {
		t.Fatal(err)
	}
	if stale.RequestIdentity != current.RequestIdentity ||
		stale.RequestRevisionIdentity == current.RequestRevisionIdentity {
		t.Fatal("R3AK stale fixture did not preserve semantic identity / change revision")
	}

	response := actorContextR3AKResponseWitnessProbe{
		RequestIdentity:         stale.RequestIdentity,
		RequestRevisionIdentity: stale.RequestRevisionIdentity,
		ResponseRef:             "test-only:r3ak/stale-response",
	}
	if _, err := actorContextR3AKCorrelateResponse(current, response); err == nil {
		t.Fatal("R3AK correlated stale request revision to current request")
	}
}

func TestActorContextR3AKRequestIdentityMismatchFailsClosed(t *testing.T) {
	record, err := actorContextR3AKPrepareRequest(actorContextR3AKCurrentRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	response := actorContextR3AKResponseWitnessProbe{
		RequestIdentity:         record.RequestIdentity + ":different",
		RequestRevisionIdentity: record.RequestRevisionIdentity,
		ResponseRef:             "test-only:r3ak/wrong-request",
	}
	if _, err := actorContextR3AKCorrelateResponse(record, response); err == nil {
		t.Fatal("R3AK correlated response for different semantic request identity")
	}
}

func TestActorContextR3AKDifferentInformationClassCanStillCorrelate(t *testing.T) {
	record, err := actorContextR3AKPrepareRequest(actorContextR3AKCurrentRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	response := actorContextR3AKResponseWitnessProbe{
		RequestIdentity:         record.RequestIdentity,
		RequestRevisionIdentity: record.RequestRevisionIdentity,
		ResponseRef:             "test-only:r3ak/partial-response",
		ProvidedInformationClasses: []string{
			"test-only:different-information-class",
		},
		EvidenceRefs: []string{"test-only:r3ak/partial-evidence"},
	}
	got, err := actorContextR3AKCorrelateResponse(record, response)
	if err != nil {
		t.Fatal(err)
	}
	if got.CorrelationStatus != actorContextR3AKCorrelationMatchedCurrent ||
		!reflect.DeepEqual(got.ProvidedInformationClasses, []string{
			"test-only:different-information-class",
		}) {
		t.Fatalf("R3AK treated information-class adequacy as correlation identity: %+v", got)
	}
}

func TestActorContextR3AKRecipientUnresolvedDoesNotBlockCorrelationOrCreateSender(t *testing.T) {
	request := actorContextR3AKCurrentRequest(t)
	if request.HandoffRecipientStatus != actorContextR3MHandoffRecipientUnresolved {
		t.Fatalf("R3AK setup expected unresolved recipient: %+v", request)
	}
	record, err := actorContextR3AKPrepareRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	response := actorContextR3AKResponseWitnessProbe{
		RequestIdentity:         record.RequestIdentity,
		RequestRevisionIdentity: record.RequestRevisionIdentity,
		ResponseRef:             "test-only:r3ak/unrouted-response",
	}
	if _, err := actorContextR3AKCorrelateResponse(record, response); err != nil {
		t.Fatalf("R3AK required recipient routing to correlate response: %v", err)
	}
}

func TestActorContextR3AKRepeatedCorrelationDoesNotStrengthenOrDuplicateAuthority(t *testing.T) {
	record, err := actorContextR3AKPrepareRequest(actorContextR3AKCurrentRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	response := actorContextR3AKResponseWitnessProbe{
		RequestIdentity:         record.RequestIdentity,
		RequestRevisionIdentity: record.RequestRevisionIdentity,
		ResponseRef:             "test-only:r3ak/repeat-response",
		ProvidedInformationClasses: []string{
			"authorization-evaluation-freshness-authority",
		},
	}

	first, err := actorContextR3AKCorrelateResponse(record, response)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3AKCorrelateResponse(record, response)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3AK repeated correlation changed result identity: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3AKCorrelationShapesHaveNoDeliveryAcceptanceOrActionShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3AKRequestRecordProbe{}),
		reflect.TypeOf(actorContextR3AKResponseWitnessProbe{}),
		reflect.TypeOf(actorContextR3AKCorrelationResultProbe{}),
	} {
		for _, forbidden := range []string{
			"Destination",
			"RoutingKey",
			"Channel",
			"Recipient",
			"RecipientID",
			"Sender",
			"SenderAuthority",
			"TicketID",
			"DeliveryStatus",
			"Accepted",
			"Approved",
			"Sufficient",
			"Resolved",
			"ResolutionComplete",
			"AuthorityValue",
			"AcceptedAuthorityValue",
			"Apply",
			"AutoApply",
			"ToolCall",
			"Command",
			"NextAction",
			"CanExecute",
			"Authorized",
			"Allowed",
			"Retry",
			"Reauthorize",
			"Confidence",
			"Priority",
			"Severity",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AK shape introduced forbidden delivery/acceptance/action shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3AKCurrentRequest(t *testing.T) actorContextR3AJAuthorityInformationRequestProbe {
	t.Helper()
	request, err := actorContextR3AJBuildInformationRequest(
		actorContextR3AJFreshnessReasoning(t),
		actorContextR3AJFreshnessStopView(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func actorContextR3AKPrepareRequest(
	request actorContextR3AJAuthorityInformationRequestProbe,
) (actorContextR3AKRequestRecordProbe, error) {
	if request.RequestClass == "" ||
		request.BusinessAddress == "" ||
		request.StopStage == "" ||
		request.StopClass == "" ||
		len(request.InformationNeedClasses) == 0 ||
		len(request.EvidenceNeedClasses) == 0 {
		return actorContextR3AKRequestRecordProbe{}, fmt.Errorf("R3AK request identity scope is incomplete")
	}

	record := actorContextR3AKRequestRecordProbe{
		RequestIdentity:         actorContextR3AKRequestIdentity(request),
		RequestRevisionIdentity: actorContextR3AKRequestRevisionIdentity(request),
		LifecycleState:          actorContextR3AKLifecyclePrepared,
		Request:                 actorContextR3AKNormalizeRequest(request),
		AuthorityLimits: []string{
			"prepared-does-not-mean-sent-delivered-routed-or-assigned",
			"request-identity-does-not-select-recipient-or-transport",
			"request-revision-does-not-decide-authority-freshness",
			"request-record-does-not-authorize-action",
		},
	}
	sort.Strings(record.AuthorityLimits)
	return record, nil
}

func actorContextR3AKRequestIdentity(
	request actorContextR3AJAuthorityInformationRequestProbe,
) string {
	return strings.Join([]string{
		request.RequestClass,
		request.BusinessAddress,
		request.StopStage,
		request.StopClass,
		strings.Join(actorContextR0SortedUnique(append([]string(nil), request.MachineReasons...)), ","),
		strings.Join(actorContextR0SortedUnique(append([]string(nil), request.InformationNeedClasses...)), ","),
		strings.Join(actorContextR0SortedUnique(append([]string(nil), request.EvidenceNeedClasses...)), ","),
	}, "||")
}

func actorContextR3AKRequestRevisionIdentity(
	request actorContextR3AJAuthorityInformationRequestProbe,
) string {
	return strings.Join([]string{
		actorContextR3AKRequestIdentity(request),
		strings.Join(actorContextR0SortedUnique(append([]string(nil), request.KnownSourceRefs...)), ","),
		strings.Join(actorContextR0SortedUnique(append([]string(nil), request.DrillDownRefs...)), ","),
		request.HandoffRecipientStatus,
	}, "||revision||")
}

func actorContextR3AKNormalizeRequest(
	request actorContextR3AJAuthorityInformationRequestProbe,
) actorContextR3AJAuthorityInformationRequestProbe {
	out := request
	out.MachineReasons = actorContextR0SortedUnique(append([]string(nil), request.MachineReasons...))
	out.InformationNeedClasses = actorContextR0SortedUnique(append([]string(nil), request.InformationNeedClasses...))
	out.EvidenceNeedClasses = actorContextR0SortedUnique(append([]string(nil), request.EvidenceNeedClasses...))
	out.KnownSourceRefs = actorContextR0SortedUnique(append([]string(nil), request.KnownSourceRefs...))
	out.DrillDownRefs = actorContextR0SortedUnique(append([]string(nil), request.DrillDownRefs...))
	out.AuthorityLimits = actorContextR0SortedUnique(append([]string(nil), request.AuthorityLimits...))
	return out
}

func actorContextR3AKCorrelateResponse(
	record actorContextR3AKRequestRecordProbe,
	response actorContextR3AKResponseWitnessProbe,
) (actorContextR3AKCorrelationResultProbe, error) {
	if response.ResponseRef == "" {
		return actorContextR3AKCorrelationResultProbe{}, fmt.Errorf("R3AK response ref is required")
	}
	if response.RequestIdentity != record.RequestIdentity {
		return actorContextR3AKCorrelationResultProbe{}, fmt.Errorf("R3AK semantic request identity mismatch")
	}
	if response.RequestRevisionIdentity != record.RequestRevisionIdentity {
		return actorContextR3AKCorrelationResultProbe{}, fmt.Errorf("R3AK request revision identity mismatch")
	}

	out := actorContextR3AKCorrelationResultProbe{
		RequestIdentity:         record.RequestIdentity,
		RequestRevisionIdentity: record.RequestRevisionIdentity,
		ResponseRef:             response.ResponseRef,
		CorrelationStatus:       actorContextR3AKCorrelationMatchedCurrent,
		LifecycleState:          actorContextR3AKLifecycleResponseCorrelated,
		ProvidedInformationClasses: actorContextR0SortedUnique(
			append([]string(nil), response.ProvidedInformationClasses...),
		),
		EvidenceRefs: actorContextR0SortedUnique(
			append([]string(nil), response.EvidenceRefs...),
		),
		AuthorityLimits: []string{
			"response-correlated-does-not-mean-response-accepted",
			"response-correlated-does-not-mean-evidence-sufficient",
			"response-correlated-does-not-resolve-stop",
			"provided-information-class-does-not-qualify-authority-evidence",
			"recipient-status-does-not-validate-sender-authority",
		},
	}
	sort.Strings(out.AuthorityLimits)
	return out, nil
}

func actorContextR3AKReverseStrings(values []string) []string {
	out := append([]string(nil), values...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
