package main

import (
	"fmt"
	"os"

	"tei/internal/command/lifecyclesqlite"
)

func main() {
	store, err := lifecyclesqlite.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer store.Close()
	if _, err := store.Dispatch(os.Args[2], "owner-a", 1); err != nil {
		panic(err)
	}
	if err := store.MarkUnknown(os.Args[2], "owner-a", 1, "response lost"); err != nil {
		panic(err)
	}
	fmt.Println("prepared")
}
