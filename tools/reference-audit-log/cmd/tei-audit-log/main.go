package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	auditlog "tei/tools/reference-audit-log"
)

// tei-audit-log appends one entry or verifies the hash chain of a Reference
// audit log. It is a foundation, not a production audit store.
func main() {
	if len(os.Args) < 2 || (os.Args[1] != "append" && os.Args[1] != "verify") {
		fmt.Fprintln(os.Stderr, "usage: tei-audit-log <append|verify> -log FILE [options]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	logPath := fs.String("log", "", "audit log file")
	tenant := fs.String("tenant", "", "tenant id")
	actor := fs.String("actor", "", "actor")
	action := fs.String("action", "", "action")
	subject := fs.String("subject", "", "subject")
	_ = fs.Parse(os.Args[2:])
	if *logPath == "" {
		fail(fmt.Errorf("-log is required"))
	}
	log, err := auditlog.Open(*logPath)
	if err != nil {
		fail(err)
	}
	switch command {
	case "append":
		entry, err := log.Append(*tenant, *actor, *action, *subject)
		if err != nil {
			fail(err)
		}
		emit(entry)
	case "verify":
		if err := log.Verify(); err != nil {
			fail(err)
		}
		entries, err := log.Entries()
		if err != nil {
			fail(err)
		}
		emit(map[string]any{"verified": true, "entries": len(entries)})
	}
}

func emit(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
