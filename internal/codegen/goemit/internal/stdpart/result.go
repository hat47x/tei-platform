package stdpart

// Result は、規則を確かめた答えです。
type Result string

const (
	// Satisfied は、値が規則を満たすことを表します。
	Satisfied Result = "satisfied"
	// NotSatisfied は、値があり、規則を満たさないことを表します。
	NotSatisfied Result = "not-satisfied"
	// Undecided は、値がないため規則を判定できないことを表します。
	Undecided Result = "undecided"
)
