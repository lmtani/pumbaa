package chat

// ModalKind identifies which modal is currently active.
type ModalKind int

const (
	ModalNone ModalKind = iota
	ModalSessions
)
