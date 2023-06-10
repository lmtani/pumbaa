package main

import "os"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	buildInfo := Build{
		Version: version,
		Commit:  commit,
		Date:    date,
	}
	os.Exit(Run(&buildInfo))
}
