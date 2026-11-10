package currencyrate

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
)

// RoleGranularity is the optional role that names the granularity in the
// declaration, so a table says whether it holds monthly or annual rates.
//
// Reading it from the rows alone left two gaps: an empty table had no
// granularity at all, and a table whose rows were all annual could not be told
// apart from a monthly table that had not been filled yet. X-TEA kept the two
// tables apart by naming them in system variables (D:Session.java L2253–2313),
// which said which table to read but nothing about what it held.
const RoleGranularity = "currency-rate-granularity"

// GranularityDeclaration is the granularity a rate Data declares in its rows.
type GranularityDeclaration struct {
	DataID    core.CanonicalID
	Attribute core.CanonicalID
}

// ResolveGranularity reads the granularity role from the Data declaration.
//
// A Data that does not declare the role leaves the granularity to the rows, which
// is what the reference implementation did before the role existed. A Data that
// declares the role must carry a known value in every row, which GranularityRows
// reads.
func ResolveGranularity(world ir.World, dataID core.CanonicalID) (GranularityDeclaration, error) {
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			if data.ID != dataID {
				continue
			}
			for _, attribute := range data.Attributes {
				if attribute.Role == RoleGranularity {
					return GranularityDeclaration{DataID: dataID, Attribute: attribute.ID}, nil
				}
			}
			return GranularityDeclaration{DataID: dataID}, nil
		}
	}
	return GranularityDeclaration{}, fmt.Errorf("%w: exchange rate data %s is not declared", ErrInvalidRate, dataID)
}

// GranularityRows reads the declared granularity from the rows.
//
// Every row must carry the same value: one Data describes one granularity, and two
// values in one table would mean the answer depends on which row is read first.
func (d GranularityDeclaration) GranularityRows(rows []map[core.CanonicalID]string) (Granularity, bool, error) {
	if d.Attribute == "" {
		return "", false, nil
	}
	declared := Granularity("")
	for index, row := range rows {
		value := row[d.Attribute]
		switch Granularity(value) {
		case "":
			return "", false, fmt.Errorf("exchange rate row %d: %s is empty", index+1, d.Attribute)
		case GranularityMonthly, GranularityAnnual:
		default:
			return "", false, fmt.Errorf("exchange rate row %d: %s: %q is not one of %q or %q",
				index+1, d.Attribute, value, GranularityMonthly, GranularityAnnual)
		}
		if declared != "" && declared != Granularity(value) {
			return "", false, fmt.Errorf("exchange rate rows disagree on the granularity: %q and %q", declared, value)
		}
		declared = Granularity(value)
	}
	if declared == "" {
		return "", false, nil
	}
	return declared, true, nil
}

// CheckDeclared compares a declared granularity with the one the rows imply, and
// reports a declaration that the rows contradict.
//
// The check matters because the two can disagree in a way neither alone shows: a
// table declared annual whose rows use month sequences 1..12 would be read as
// monthly by a caller that trusts the rows and as annual by a caller that trusts
// the declaration.
func CheckDeclared(declared Granularity, implied Granularity) error {
	if declared == "" {
		return nil
	}
	if declared != implied {
		return fmt.Errorf("%w: the table declares %s but its rows are %s", ErrInvalidRate, declared, implied)
	}
	return nil
}
