package ports

import "github.com/lmtani/pumbaa/internal/types"

type Writer interface {
	Primary(string)
	Accent(string)
	Message(string)
	Error(string)
	Table(table types.Table)
	QueryTable(table types.QueryResponse)
	ResourceTable(table types.TotalResources)
	MetadataTable(d types.MetadataResponse) error
	Json(interface{}) error
}
