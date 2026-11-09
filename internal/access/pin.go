package access

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HeadPin is where the last hash of a chained journal is kept.
//
// It exists because the chain alone cannot detect removal of the last record: a
// shortened chain is still a valid chain. Keeping the head somewhere the journal
// cannot rewrite turns that case into a mismatch.
//
// The pin is not a signature, and it protects only as far as its own file is
// protected. A party able to rewrite both the journal and the pin can make them
// agree. What the pin adds is that a partial change — a journal edited or
// truncated without also updating the pin — no longer passes unnoticed, and a
// pin kept outside the journal's directory (a tag, a release note, a separate
// store) extends that to a party who can only write the journal.
type HeadPin struct {
	Path string
}

// Read returns the pinned head, or the empty string when nothing is pinned yet.
// A missing pin is not an error: a journal that has never been pinned has no
// head to compare against, and refusing that would make the first append
// impossible.
func (p HeadPin) Read() (string, error) {
	if p.Path == "" {
		return "", nil
	}
	raw, err := os.ReadFile(p.Path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read head pin %s: %w", p.Path, err)
	}
	lines := 0
	var head string
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines++
		head = line
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read head pin %s: %w", p.Path, err)
	}
	if lines > 1 {
		return "", fmt.Errorf("head pin %s holds %d heads, so it does not pin one journal", p.Path, lines)
	}
	return head, nil
}

// Write replaces the pinned head. The file is written to a temporary name and
// renamed, so a reader never sees a half-written pin; a torn pin would report a
// mismatch that never happened.
func (p HeadPin) Write(head string) error {
	if p.Path == "" {
		return fmt.Errorf("no head pin path is configured")
	}
	if head == "" {
		return fmt.Errorf("no head is given to pin")
	}
	if err := os.MkdirAll(filepath.Dir(p.Path), 0o755); err != nil {
		return fmt.Errorf("prepare head pin %s: %w", p.Path, err)
	}
	temporary := p.Path + ".tmp"
	if err := os.WriteFile(temporary, []byte(head+"\n"), 0o600); err != nil {
		return fmt.Errorf("write head pin %s: %w", p.Path, err)
	}
	if err := os.Rename(temporary, p.Path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("replace head pin %s: %w", p.Path, err)
	}
	return nil
}

// PinnedJournal appends to a chained journal while keeping its head pinned.
//
// It implements Journal, so a caller that already takes a journal only has to
// pick this one instead of a plain file to get the suffix check as well.
type PinnedJournal struct {
	Journal ChainFile
	Pin     HeadPin
}

// Append checks the pin against the journal, appends, and pins the new head.
//
// The order matters: the pin is checked before the append, because an append to
// a journal that was truncated would extend a chain the reader cannot trust; and
// the new head is pinned after the append, because pinning a head the journal
// does not have would report a mismatch on the next run for no reason.
func (j PinnedJournal) Append(record Record) error {
	if j.Pin.Path == "" {
		return fmt.Errorf("no head pin is configured, so the journal cannot be checked for removal")
	}
	if err := j.checkPin(); err != nil {
		return err
	}
	if err := j.Journal.Append(record); err != nil {
		return err
	}
	head, err := j.Journal.Head()
	if err != nil {
		return err
	}
	if err := j.Pin.Write(head); err != nil {
		return fmt.Errorf("pin the journal head: %w", err)
	}
	return nil
}

// checkPin compares the pin with the journal.
func (j PinnedJournal) checkPin() error {
	pinned, err := j.Pin.Read()
	if err != nil {
		return err
	}
	if pinned == "" {
		// Nothing pinned yet: accept a journal that is internally consistent, so
		// the first run over an existing journal works. From then on the pin
		// covers the suffix.
		return j.Journal.Verify()
	}
	return j.Journal.VerifyHead(pinned)
}

// Verify checks the journal and the pin together. It is what a reader calls
// before trusting the journal: the chain proves nothing was edited, and the pin
// proves nothing was removed from the end.
func (j PinnedJournal) Verify() error {
	if j.Pin.Path == "" {
		return fmt.Errorf("no head pin is configured, so removal of the last record is not checked")
	}
	pinned, err := j.Pin.Read()
	if err != nil {
		return err
	}
	return j.Journal.VerifyHead(pinned)
}
