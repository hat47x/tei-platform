//go:build !tei_repair_only

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"tei/internal/security/effectcommit/submit"
)

// cliOrderRun is a TEST-ONLY, in-process stand-in for a second consumer of the
// target-neutral submit.Caller: `submit-order <id>` and `lookup-order <id>`. It
// has no process or HTTP boundary and is not a real command-line product. It
// deliberately does NOT use runnerOrderSubmitStore, HTTP, hostEffect or
// ReadStatus; it builds its own Spec (a different Application and Capability
// from the Web store) and its own payload (different key order, so the bytes
// and hash differ from the Web store's).
//
// It gives Replayed a different user-facing meaning from the Web store: the
// Web store refuses a repeated submit (errOrderNotSubmittable), while this
// driver exits 0 with "already submitted". The neutral layer does not force
// either choice.
//
// Exit codes, one stable line per Disposition:
//
//	0 committed    "submitted order N operation=ID"
//	0 replayed     "already submitted order N operation=ID"
//	3 not-committed "not submitted order N (denied|resource-conflict)"
//	4 uncertain    "unconfirmed order N; run lookup-order N"
//	5 failed       "failed order N"
//	6 not-recorded "no record for order N"
//	2 usage error  "usage: submit-order <id> | lookup-order <id>"
const (
	cliExitOK             = 0
	cliExitUsage          = 2
	cliExitNotCommitted   = 3
	cliExitUncertain      = 4
	cliExitFailed         = 5
	cliExitNotRecorded    = 6
	cliApplication        = "order_entry_cli"
	cliCapability         = "order_entry_cli.submit_order"
	cliOperationIDPattern = "cli-order-submit/%d"
)

// cliSpec installs the same shared intent check as the Web store (D11).
var cliSpec = submit.Spec{Application: cliApplication, Capability: cliCapability, ValidateIntent: checkOrderSubmitIntent}

func cliIntent(orderID int64) submit.Intent {
	return submit.Intent{
		OperationID:     fmt.Sprintf(cliOperationIDPattern, orderID),
		ExpectedVersion: 0,
		Payload:         []byte(fmt.Sprintf(`{"to":"submitted","orderId":%d}`, orderID)),
	}
}

func cliOrderRun(ctx context.Context, caller submit.Caller, args []string, stdout io.Writer) int {
	usage := func() int {
		fmt.Fprintln(stdout, "usage: submit-order <id> | lookup-order <id>")
		return cliExitUsage
	}
	if len(args) != 2 || (args[0] != "submit-order" && args[0] != "lookup-order") {
		return usage()
	}
	id, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil || id <= 0 {
		return usage()
	}
	grant, err := caller.Grant(ctx)
	if err != nil {
		fmt.Fprintf(stdout, "failed order %d\n", id)
		return cliExitFailed
	}
	var out submit.Outcome
	if args[0] == "submit-order" {
		out, err = caller.Execute(ctx, grant, cliIntent(id))
	} else {
		out, err = caller.Lookup(ctx, grant, cliIntent(id))
	}
	if errors.Is(err, submit.ErrIntentRejected) {
		fmt.Fprintf(stdout, "not submitted order %d (rejected)\n", id)
		return cliExitNotCommitted
	}
	if err != nil {
		fmt.Fprintf(stdout, "failed order %d\n", id)
		return cliExitFailed
	}
	switch out.Disposition {
	case submit.Committed:
		fmt.Fprintf(stdout, "submitted order %d operation=%s\n", id, out.OperationID)
		return cliExitOK
	case submit.Replayed:
		fmt.Fprintf(stdout, "already submitted order %d operation=%s\n", id, out.OperationID)
		return cliExitOK
	case submit.NotCommitted:
		fmt.Fprintf(stdout, "not submitted order %d (%s)\n", id, out.Status)
		return cliExitNotCommitted
	case submit.Uncertain:
		fmt.Fprintf(stdout, "unconfirmed order %d; run lookup-order %d\n", id, id)
		return cliExitUncertain
	case submit.NotRecorded:
		fmt.Fprintf(stdout, "no record for order %d\n", id)
		return cliExitNotRecorded
	default:
		fmt.Fprintf(stdout, "failed order %d\n", id)
		return cliExitFailed
	}
}
