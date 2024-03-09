package ports

import "github.com/lmtani/pumbaa/internal/types"

type Writer interface {
	Primary(string)
	Accent(string)
	Error(string)
	Table(table types.Table)
}
