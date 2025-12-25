// Package workflow contains domain entities and types.
package workflow

// CallLog represents log information for a call.
type CallLog struct {
	Stdout     string
	Stderr     string
	Attempt    int
	ShardIndex int
}
