//go:build !tei_order_only

package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/ir"
)

// errCustomerLookupInput marks input the Target itself rejects before any Driver call.
var errCustomerLookupInput = errors.New("invalid customer lookup input")

// referenceCustomerLookupScenarioRunner runs the second Application's customer reference Pattern
// through the Driver. It reuses the customer meaning (customer.customer) and the Contract Query
// that order_entry already uses; it does not copy either.
type referenceCustomerLookupScenarioRunner struct {
	resolver      driver.Resolver
	environmentID core.CanonicalID
	applicationID core.CanonicalID
	patternID     core.CanonicalID
	contracts     driver.ContractQueryExecutorRegistry
}

func newReferenceCustomerLookupScenarioRunner(world ir.World) *referenceCustomerLookupScenarioRunner {
	return &referenceCustomerLookupScenarioRunner{
		resolver:      driver.NewResolver(world),
		environmentID: core.MustCanonicalID("development"),
		applicationID: core.MustCanonicalID("repair_intake"),
		patternID:     core.MustCanonicalID("repair_intake.customer_reference"),
		contracts:     referenceCustomerContractQueryExecutors(),
	}
}

// ApplicationID returns the Application this runner executes (see referenceReadScenarioRunner).
func (r *referenceCustomerLookupScenarioRunner) ApplicationID() string {
	return string(r.applicationID)
}

func (r *referenceCustomerLookupScenarioRunner) RunCustomerLookup(
	ctx context.Context,
	input customerLookupInput,
) (customerLookupResult, error) {
	raw := strings.TrimSpace(input.CustomerID)
	if raw == "" {
		return customerLookupResult{}, fmt.Errorf("%w: customerId is required", errCustomerLookupInput)
	}
	// The Reference executor and the order_entry browse fixture carry customer ids as int64.
	// This conversion is a Target-local fixture choice, not a Canonical type rule.
	customerID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return customerLookupResult{}, fmt.Errorf("%w: customerId must be a base-10 integer", errCustomerLookupInput)
	}

	// The reference Pattern reads its key from the Relation's source attribute. Build that one
	// mapped row from the input instead of guessing a column from a Query result.
	intake := queryresult.MappedResult{
		Columns: []queryresult.Column{{
			Raw:       "customer_id",
			Attribute: core.MustCanonicalID("repair_intake.intake.customer_id"),
		}},
		Rows: [][]any{{customerID}},
	}

	customer, err := r.resolver.ExecuteReferenceContractQueryInEnvironment(
		ctx,
		r.environmentID,
		r.applicationID,
		r.patternID,
		intake,
		0,
		r.contracts,
	)
	if err != nil {
		return customerLookupResult{}, err
	}
	rows := projectContractRows(customer.Execution.Result.Rows)
	if len(rows) != 1 {
		return customerLookupResult{}, fmt.Errorf(
			"customer reference returned %d rows for a Relation that targets one customer",
			len(rows),
		)
	}
	intakeRows, err := projectMappedRows(intake)
	if err != nil {
		return customerLookupResult{}, err
	}

	return customerLookupResult{
		ScenarioID:    "repair-intake.customer-lookup-web-slice",
		Environment:   string(r.environmentID),
		ApplicationID: string(r.applicationID),
		CustomerID:    strconv.FormatInt(customerID, 10),
		Intake:        intakeRows[0],
		CustomerRows:  rows,
		RuntimeBindings: []runtimeBindingProjection{
			projectEnvironmentBinding(string(r.patternID), customer.Environment.ID, customer.Executor),
		},
		AuthorityLimits: []string{
			"synthetic-runtime-fixture-is-not-canonical-data",
			"executor-is-a-test-double-not-a-real-database-connection",
			"customer-meaning-and-contract-query-are-reused-from-order-entry-not-copied",
			"lookup-does-not-imply-authorization-or-command-readiness",
			"http-interaction-does-not-define-semantic-outcome",
		},
		Unresolved: []string{
			"standalone-lookup-endpoint-is-reference-diagnostic-without-authentication-gate",
			"production-environment-has-no-contract-query-executor-binding",
			"actual-oracle-principal-credential-network-authority-not-observed",
			"customer-id-integer-conversion-is-target-local-fixture-choice",
			"actual-functional-evidence-not-yet-acquired",
			"visual-evidence-not-yet-acquired",
		},
	}, nil
}

