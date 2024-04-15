package cromwell

import (
	"testing"

	"github.com/lmtani/pumbaa/internal/adapters/test"

	"github.com/lmtani/pumbaa/internal/adapters"
	"github.com/lmtani/pumbaa/internal/types"
)

func TestKill_Kill(t *testing.T) {
	expected := types.SubmitResponse{
		ID:     "fake-id",
		Status: "fake-status",
	}
	fakeCromwell := test.FakeCromwell{
		SubmitResponse: expected,
	}
	l := adapters.NewLogger(adapters.InfoLevel)

	i := NewCromwell(&fakeCromwell, l)

	resp, err := i.Kill("fake-id")
	if err != nil {
		t.Error(err)
	}

	if resp != expected {
		t.Errorf("Expected %v, got %v", expected, i)
	}

	if !fakeCromwell.AbortCalled {
		t.Errorf("Expected MetadataCalled to be true, found %v", fakeCromwell.MetadataCalled)
	}

}
