package validationeval_test

import (
	"math/big"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// numbersOnlyRows reads numbers and nothing else, so a reader that cannot read
// the declared type over rows is available to a test.
type numbersOnlyRows struct{}

func (numbersOnlyRows) ReadTypedRows(input semantic.OperatorInputDeclaration, relation, attribute core.CanonicalID) (validationeval.TypedRowReading, error) {
	return validationeval.TypedRowReading{}, nil
}

func (numbersOnlyRows) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: new(big.Rat)}, nil
}

func (numbersOnlyRows) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}
