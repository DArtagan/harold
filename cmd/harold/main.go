// Command harold is a Todoist UI built for triage.
package main

import (
	"fmt"
	"os"
)

// version is set at build time from the release tag; see docs/releasing.md.
var version = "dev"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}
	fmt.Fprintln(os.Stderr, "harold: the triage UI is not built yet; see todos/README.md")
	os.Exit(1)
}
