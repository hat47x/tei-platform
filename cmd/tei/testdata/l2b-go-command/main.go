// Command l2b-go-command is a minimal build subject for the L2-b regression
// test in cmd/tei. It exists so the chain can run a real Go build without paying
// for the whole internal tree; the mechanism under test is the same.
package main

import "fmt"

func main() {
	fmt.Println("l2b go-command fixture")
}
