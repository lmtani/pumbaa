package ports

import "github.com/lmtani/pumbaa/internal/types"

type CromwellServer interface {
	Kill(o string) (types.SubmitResponse, error)
	Status(o string) (types.SubmitResponse, error)
	Outputs(o string) (types.OutputsResponse, error)
	Query(params *types.ParamsQueryGet) (types.QueryResponse, error)
	Metadata(o string, params *types.ParamsMetadataGet) (types.MetadataResponse, error)
	Submit(wdl, inputs, dependencies, options string) (types.SubmitResponse, error)
}
