package cromwell

import (
	"testing"

	"github.com/lmtani/pumbaa/internal/adapters/logger"

	"github.com/lmtani/pumbaa/internal/adapters/test"

	"github.com/lmtani/pumbaa/internal/types"
)

var submitResponse = types.SubmitResponse{
	ID:     "fake-id",
	Status: "fake-status",
}

func TestKill_Kill(t *testing.T) {
	fakeCromwell := test.FakeCromwell{
		SubmitResponse: submitResponse,
	}
	l := logger.NewLogger(logger.InfoLevel)

	i := NewCromwell(&fakeCromwell, l)

	resp, err := i.Kill("fake-id")
	if err != nil {
		t.Error(err)
	}

	if resp != submitResponse {
		t.Errorf("Expected %v, got %v", submitResponse, i)
	}

	if !fakeCromwell.AbortCalled {
		t.Errorf("Expected MetadataCalled to be true, found %v", fakeCromwell.MetadataCalled)
	}
}
