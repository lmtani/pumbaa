package ports

// ProgressReporter surfaces the stages of a long-running operation while it
// runs. It exists because some analyses spend most of their wall clock waiting
// on remote metadata, and silence for a minute reads as a hang.
//
// Implementations must tolerate being nil-checked away: an operation that
// reports nothing must behave identically to one that does.
type ProgressReporter interface {
	// Step announces what is being done now, replacing whatever came before.
	Step(format string, args ...any)
	// Done clears the reporter's output so the result is not printed under a
	// stale progress line.
	Done()
}
