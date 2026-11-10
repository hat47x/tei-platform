// Command probe is a build subject with no module-local dependencies, so a plan
// that names only this package covers everything the build compiles.
package main

import "fmt"

func main() {
	fmt.Println("probe")
}
