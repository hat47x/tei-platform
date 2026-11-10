package main

// argumentError marks a failure caused by the caller's arguments: a required
// flag was not supplied, a flag was repeated, or a flag combination is not
// allowed. It exists so every command answers the same condition with the same
// stable code (#3733).
//
// A command must not reuse a semantic code for this. `INVALID_SUBJECT` means
// "the subject value is not usable" (empty, whitespace-only, not a canonical
// id); leaving out `-workspace` is an argument problem, not a subject problem.
// Read the distinction when adding a new guard: if the fix is "supply the flag",
// it is an argument error.
type argumentError struct {
	message string
}

func (e *argumentError) Error() string { return e.message }

// newArgumentError builds one. The message should name the flags that were
// missing, the same way the other commands name theirs.
func newArgumentError(message string) error { return &argumentError{message: message} }
