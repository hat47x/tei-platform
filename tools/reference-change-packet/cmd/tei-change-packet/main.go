package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	changepacket "tei/tools/reference-change-packet"
)

// The CLI validates a packet, checks the changed paths, or runs the declared
// verification. It never edits the packet, the Workspace, or Git state.
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tei-change-packet <validate|scope|verify> -packet FILE [-repo DIR]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	packetPath := fs.String("packet", "", "change packet JSON")
	repo := fs.String("repo", ".", "repository root")
	_ = fs.Parse(os.Args[2:])
	if *packetPath == "" {
		fail(fmt.Errorf("-packet is required"))
	}
	packet, err := changepacket.Load(*packetPath)
	if err != nil {
		fail(err)
	}

	switch command {
	case "validate":
		emit(packet)
	case "scope":
		base := packet.Base
		if base == "" {
			base = "HEAD"
		}
		changed, err := changepacket.ChangedFiles(context.Background(), *repo, base)
		if err != nil {
			fail(err)
		}
		violations := changepacket.ScopeViolations(changed, packet)
		emit(map[string]any{
			"packetId":        packet.ID,
			"base":            base,
			"changedFiles":    changed,
			"scopeViolations": violations,
			"inScope":         len(violations) == 0,
		})
		if len(violations) > 0 {
			os.Exit(1)
		}
	case "verify":
		receipt, err := changepacket.Run(context.Background(), packet, *repo)
		if err != nil {
			fail(err)
		}
		emit(receipt)
		if !receipt.Passed {
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", command)
		os.Exit(2)
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
