package debug

// ModalKind identifies which modal is currently active.
type ModalKind int

const (
	ModalNone ModalKind = iota
	ModalChatSelection
	ModalHelp
	ModalLog
	ModalInputs
	ModalOutputs
	ModalOptions
	ModalGlobalTimeline
	ModalCallInputs
	ModalCallOutputs
	ModalCallCommand
	ModalBatchLogs
	ModalCopyMenu
	ModalFailureSummary
	ModalError
)
