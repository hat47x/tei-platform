package businesscalendar

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// ValueDomain is the set of calendar identities a holiday Data admits, resolved
// from the declaration of the attribute that carries the calendar identity.
//
// X-TEA compared a calendar code string against a "code;date" list entry
// (D:Session.java L786, L1691), so a code that no calendar declared still
// matched rows and no one noticed. Here the identity attribute can declare its
// values as a type domain, and a row outside it is reported.
type ValueDomain struct {
	// Declared is true when the identity attribute's type declares values. A
	// type that declares none is not read as "no calendar is allowed".
	Declared bool
	Values   []string
}

// CalendarDomain resolves the values the holiday identity attribute admits.
//
// The domain comes from the attribute's type: a type that declares `values`
// lists them, and an alias to such a type uses the same list. A type that
// declares no values yields Declared=false, and the caller must not treat that
// as a closed domain.
func (b HolidayBinding) CalendarDomain(world ir.World) (ValueDomain, error) {
	if b.Calendar == "" {
		return ValueDomain{}, nil
	}
	data, ok := dataByID(world, b.DataID)
	if !ok {
		return ValueDomain{}, fmt.Errorf("holiday data %s is not declared", b.DataID)
	}
	var typeRef core.CanonicalID
	for _, attribute := range data.Attributes {
		if attribute.ID == b.Calendar {
			typeRef = attribute.TypeRef
			break
		}
	}
	if typeRef == "" {
		return ValueDomain{}, fmt.Errorf("holiday data %s does not declare %s", b.DataID, b.Calendar)
	}
	values, declared, ok := declaredValues(world, typeRef)
	if !ok {
		return ValueDomain{}, fmt.Errorf("holiday data %s calendar attribute %s has no resolvable type", b.DataID, b.Calendar)
	}
	return ValueDomain{Declared: declared, Values: values}, nil
}

// Validate checks one row's calendar identity against the domain. It reports a
// row whose identity is outside the declared values, so a typo in the identity
// does not silently lose the holiday.
func (d ValueDomain) Validate(index int, identity string) error {
	if !d.Declared {
		return nil
	}
	if identity == "" {
		return fmt.Errorf("holiday row %d: the calendar identity is empty", index+1)
	}
	for _, value := range d.Values {
		if value == identity {
			return nil
		}
	}
	return fmt.Errorf("holiday row %d: calendar identity %q is not one of %v", index+1, identity, d.Values)
}

// declaredValues follows the alias chain to the type that declares values.
// A type that declares no values returns declared=false and ok=true, because
// the type resolved but says nothing about a closed set.
func declaredValues(world ir.World, typeRef core.CanonicalID) ([]string, bool, bool) {
	types := map[core.CanonicalID]semantic.Type{}
	for _, namespace := range world.Semantics {
		for _, typ := range namespace.Types {
			types[typ.ID] = typ
		}
	}
	current := typeRef
	for depth := 0; depth < 16; depth++ {
		typ, ok := types[current]
		if !ok {
			return nil, false, false
		}
		if len(typ.Values) > 0 {
			return append([]string(nil), typ.Values...), true, true
		}
		if typ.Kind == semantic.TypeAlias && typ.Underlying != "" {
			current = typ.Underlying
			continue
		}
		return nil, false, true
	}
	return nil, false, false
}
