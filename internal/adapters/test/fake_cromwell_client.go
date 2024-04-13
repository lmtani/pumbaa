package test

import "github.com/lmtani/pumbaa/internal/types"

type FakeCromwell struct {
	MetadataCalled bool
	AbortCalled    bool
	StatusCalled   bool
	OutputsCalled  bool
	QueryCalled    bool
	SubmitCalled   bool

	SubmitResponse   types.SubmitResponse
	MetadataResponse types.MetadataResponse
	OutputsResponse  types.OutputsResponse
	QueryResponse    types.QueryResponse

	Err error
}

func (f *FakeCromwell) Metadata(o string, p *types.ParamsMetadataGet) (types.MetadataResponse, error) {
	f.MetadataCalled = true
	return f.MetadataResponse, f.Err
}

func (f *FakeCromwell) Kill(o string) (types.SubmitResponse, error) {
	f.AbortCalled = true
	return f.SubmitResponse, f.Err
}

func (f *FakeCromwell) Status(o string) (types.SubmitResponse, error) {
	f.StatusCalled = true
	return f.SubmitResponse, f.Err
}

func (f *FakeCromwell) Outputs(o string) (types.OutputsResponse, error) {
	f.OutputsCalled = true
	return f.OutputsResponse, f.Err
}

func (f *FakeCromwell) Query(p *types.ParamsQueryGet) (types.QueryResponse, error) {
	f.QueryCalled = true
	return f.QueryResponse, f.Err
}

func (f *FakeCromwell) Submit(requestFields *types.SubmitRequest) (types.SubmitResponse, error) {
	f.SubmitCalled = true
	return f.SubmitResponse, f.Err
}
