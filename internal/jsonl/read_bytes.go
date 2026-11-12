package jsonl

import (
	"bufio"
	"bytes"
	"fmt"
)

// ReadBytes decodes JSONL records from already acquired bytes. It applies the
// same line-size limit, blank-line rule and prefix-on-error behavior as Read.
//
// External Evidence receivers can fingerprint the original bytes once and
// inspect them without opening the source again. This function has no file or
// network effects and does not treat a partial stream as a complete delivery.
func ReadBytes[T any](raw []byte, label string, decode func([]byte) (T, error)) ([]T, error) {
	var records []T
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		text := scanner.Bytes()
		if len(text) == 0 {
			continue
		}
		record, err := decode(text)
		if err != nil {
			return records, fmt.Errorf("%s bytes line %d: %w", label, line, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return records, fmt.Errorf("read %s bytes: %w", label, err)
	}
	return records, nil
}
