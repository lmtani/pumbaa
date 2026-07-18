package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// stubQuerier returns a fixed set of prior runs.
type stubQuerier struct{ runs []domain.Workflow }

func (s *stubQuerier) Query(context.Context, domain.QueryFilter) (*domain.QueryResult, error) {
	return &domain.QueryResult{Workflows: s.runs}, nil
}

// stubFetcher serves parameter documents by run id.
type stubFetcher struct {
	workflow *domain.Workflow
	params   map[string]string
}

func (s *stubFetcher) GetRawMetadataWithOptions(context.Context, string, bool) ([]byte, error) {
	return nil, errors.New("not used")
}
func (s *stubFetcher) GetWorkflowCost(context.Context, string) (float64, string, error) {
	return 0, "", nil
}
func (s *stubFetcher) ParseMetadata([]byte) (*domain.Workflow, error) { return s.workflow, nil }
func (s *stubFetcher) GetSubmittedInputs(_ context.Context, id string) (string, error) {
	if doc, ok := s.params[id]; ok {
		return doc, nil
	}
	return "", errors.New("unreadable")
}

func referenceUseCase(runs []domain.Workflow, params map[string]string) *CacheForecastUseCase {
	reference := &domain.Workflow{ID: "chosen", Name: "W"}
	return NewCacheForecastUseCase(
		&stubMetadataReader{workflow: reference},
		&stubFetcher{workflow: reference, params: params},
		&stubQuerier{runs: runs},
		&mockFileProvider{},
		nil,
	)
}

// One candidate is not a choice, so there is nothing to defer to the user.
func TestResolveReferenceUsesTheOnlyCandidate(t *testing.T) {
	uc := referenceUseCase([]domain.Workflow{{ID: "only", Name: "W"}}, nil)

	got, err := uc.resolveReference(context.Background(), "", "W", nil)
	if err != nil {
		t.Fatalf("resolveReference() error: %v", err)
	}
	if got == nil {
		t.Fatal("expected the single candidate to be used")
	}
}

// Several candidates is a choice, and choosing it silently is what produced a
// confident, pessimistic forecast against an unrelated run.
func TestResolveReferenceRefusesToChooseBetweenSeveral(t *testing.T) {
	uc := referenceUseCase([]domain.Workflow{
		{ID: "a", Name: "W"},
		{ID: "b", Name: "W"},
	}, map[string]string{
		"a": `{"W.sample":"other"}`,
		"b": `{"W.sample":"mine"}`,
	})

	_, err := uc.resolveReference(context.Background(), "", "W",
		map[string]any{"W.sample": "mine"})

	var ambiguous *AmbiguousReferenceError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("error = %v, want an ambiguous-reference error", err)
	}
	if len(ambiguous.Candidates) != 2 {
		t.Fatalf("got %d candidates, want both", len(ambiguous.Candidates))
	}
	// The nearest run leads, so the list answers the question it poses.
	if ambiguous.Candidates[0].ID != "b" {
		t.Errorf("first candidate = %q, want b — its parameters match", ambiguous.Candidates[0].ID)
	}
	if ambiguous.Candidates[0].Differing != 0 {
		t.Errorf("nearest candidate differs in %d parameters, want 0", ambiguous.Candidates[0].Differing)
	}
}

// A run named explicitly is used whatever else exists.
func TestResolveReferenceHonoursAnExplicitChoice(t *testing.T) {
	uc := referenceUseCase([]domain.Workflow{{ID: "a"}, {ID: "b"}}, nil)

	got, err := uc.resolveReference(context.Background(), "named", "W", nil)
	if err != nil {
		t.Fatalf("resolveReference() error: %v", err)
	}
	if got == nil {
		t.Fatal("expected the named run to be fetched")
	}
}

// A candidate whose parameters cannot be read stays in the list, last: dropping
// it would claim it is not a candidate, which is more than we know.
func TestResolveReferenceKeepsUnreadableCandidatesLast(t *testing.T) {
	uc := referenceUseCase([]domain.Workflow{
		{ID: "unreadable", End: time.Now()},
		{ID: "readable"},
	}, map[string]string{"readable": `{"W.sample":"mine"}`})

	_, err := uc.resolveReference(context.Background(), "", "W",
		map[string]any{"W.sample": "mine"})

	var ambiguous *AmbiguousReferenceError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("error = %v, want an ambiguous-reference error", err)
	}
	last := ambiguous.Candidates[len(ambiguous.Candidates)-1]
	if last.ID != "unreadable" || last.Readable {
		t.Errorf("last candidate = %+v, want the unreadable one", last)
	}
}

func TestCountDifferences(t *testing.T) {
	a := map[string]any{"x": "1", "y": "2", "gone": "3"}
	b := map[string]any{"x": "1", "y": "changed", "added": "4"}

	differing, total := countDifferences(a, b)
	if total != 4 {
		t.Errorf("total = %d, want 4 distinct keys", total)
	}
	// y changed, gone is missing on one side, added on the other.
	if differing != 3 {
		t.Errorf("differing = %d, want 3", differing)
	}
}
