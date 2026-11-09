package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	offlinestorage "tei/tools/reference-offline-storage"
)

// tei-offline-storage reads and writes a local Reference store. It does not
// synchronize with a remote service.
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tei-offline-storage <put|get|delete|list> -root DIR [-key KEY] [-value TEXT]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	root := fs.String("root", "", "storage root")
	key := fs.String("key", "", "key")
	value := fs.String("value", "", "value for put")
	_ = fs.Parse(os.Args[2:])
	if *root == "" {
		fail(fmt.Errorf("-root is required"))
	}
	store, err := offlinestorage.Open(*root)
	if err != nil {
		fail(err)
	}
	defer store.Close()
	switch command {
	case "put":
		if *key == "" {
			fail(fmt.Errorf("-key is required for put"))
		}
		if err := store.Put(*key, []byte(*value)); err != nil {
			fail(err)
		}
		emit(map[string]any{"key": *key, "bytes": len(*value)})
	case "get":
		if *key == "" {
			fail(fmt.Errorf("-key is required for get"))
		}
		content, err := store.Get(*key)
		if err != nil {
			fail(err)
		}
		os.Stdout.Write(content)
	case "delete":
		if *key == "" {
			fail(fmt.Errorf("-key is required for delete"))
		}
		if err := store.Delete(*key); err != nil {
			fail(err)
		}
		emit(map[string]any{"deleted": *key})
	case "list":
		keys, err := store.List()
		if err != nil {
			fail(err)
		}
		emit(keys)
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
