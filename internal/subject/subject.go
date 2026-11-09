// Package subject is the legacy import-path adapter for the canonical Core
// subject value. Do not duplicate fields or validation in this package.
package subject

import coresubject "tei/core/subject"

// Subject is an alias, not a new type. Existing code may continue importing
// tei/internal/subject while the single source of behavior is core/subject.
type Subject = coresubject.Subject
