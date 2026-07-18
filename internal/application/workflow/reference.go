package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/lmtani/pumbaa/internal/application"
	domain "github.com/lmtani/pumbaa/internal/domain/workflow"
)

const (
	// referenceCandidates bounds how far back the search looks. Far enough to
	// cover "which run of this did I do before", short enough that a workflow
	// with years of history does not turn the search into the slow part.
	referenceCandidates = 30
	// referenceRanking bounds the concurrent parameter-document fetches. The
	// work is latency, so this is about being polite rather than about limits
	// on our side.
	referenceRanking = 16
)

// ReferenceCandidate is a prior run that could serve as the comparison point,
// with the evidence a user needs to choose between them.
type ReferenceCandidate struct {
	ID   string
	Name string
	End  time.Time
	// Differing and Total describe how far this run's parameters are from the
	// pending submission's. Total is zero when the run's parameters could not
	// be read, which is why they are reported as a pair rather than a ratio.
	Differing int
	Total     int
	Readable  bool
}

// AmbiguousReferenceError reports that several prior runs could serve as the
// reference and that choosing between them is not ours to do.
//
// It carries the candidates because the alternative — telling the user to go
// and look — reproduces the very mistake the refusal exists to avoid: a list
// ordered by date, with nothing to say which run is comparable, invites picking
// the most recent, which is exactly the guess we declined to make.
type AmbiguousReferenceError struct {
	Workflow   string
	Candidates []ReferenceCandidate
}

func (e *AmbiguousReferenceError) Error() string {
	return fmt.Sprintf("%d previous runs of %q could serve as the reference; choose one with --against",
		len(e.Candidates), e.Workflow)
}

// resolveReference finds the run to compare against.
//
// It will use a run the caller names, and it will use the only candidate when
// there is only one — neither is a choice. It will not pick between several:
// which prior run is comparable depends on what the user was doing, and a run
// of the same workflow with different parameters produces a forecast that is
// pessimistic, confident and wrong.
func (uc *CacheForecastUseCase) resolveReference(
	ctx context.Context,
	id, workflowName string,
	pendingParams map[string]any,
) (*domain.Workflow, error) {
	if id != "" {
		w, err := uc.fetchReference(ctx, id)
		if err != nil {
			return nil, application.NewUseCaseError("cache forecast", "failed to fetch reference run", err)
		}
		return w, nil
	}
	if uc.querier == nil {
		return nil, application.NewInputValidationError("against", "is required when no querier is configured")
	}

	uc.step("looking for a previous run of %s", workflowName)
	result, err := uc.querier.Query(ctx, domain.QueryFilter{
		Name:     workflowName,
		Status:   []domain.Status{domain.StatusSucceeded},
		PageSize: referenceCandidates,
	})
	if err != nil {
		return nil, application.NewUseCaseError("cache forecast", "failed to look for a reference run", err)
	}
	if result == nil || len(result.Workflows) == 0 {
		return nil, application.NewUseCaseError("cache forecast",
			fmt.Sprintf("no successful previous run of %q to compare against", workflowName), nil)
	}

	if len(result.Workflows) == 1 {
		w, err := uc.fetchReference(ctx, result.Workflows[0].ID)
		if err != nil {
			return nil, application.NewUseCaseError("cache forecast", "failed to fetch reference run", err)
		}
		return w, nil
	}

	uc.step("comparing %d previous runs", len(result.Workflows))
	return nil, &AmbiguousReferenceError{
		Workflow:   workflowName,
		Candidates: uc.rankCandidates(ctx, result.Workflows, pendingParams),
	}
}

// rankCandidates orders prior runs by how close their parameters are to the
// pending submission's, nearest first.
//
// The distance is deliberately crude — how many keys hold a different value —
// because it is presented to a human who will recognise their own run from it,
// not used to decide anything. A candidate whose parameters cannot be read
// stays in the list, last, rather than disappearing: absence would read as "not
// a candidate", which is a stronger claim than "could not tell".
func (uc *CacheForecastUseCase) rankCandidates(
	ctx context.Context,
	runs []domain.Workflow,
	pendingParams map[string]any,
) []ReferenceCandidate {
	candidates := make([]ReferenceCandidate, len(runs))
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(referenceRanking)

	for i, run := range runs {
		candidates[i] = ReferenceCandidate{ID: run.ID, Name: run.Name, End: run.End}
		group.Go(func() error {
			document, err := uc.fetcher.GetSubmittedInputs(groupCtx, run.ID)
			if err != nil {
				return nil
			}
			var params map[string]any
			if err := json.Unmarshal([]byte(document), &params); err != nil {
				return nil
			}
			candidates[i].Differing, candidates[i].Total = countDifferences(params, pendingParams)
			candidates[i].Readable = true
			return nil
		})
	}
	_ = group.Wait()

	sort.SliceStable(candidates, func(a, b int) bool {
		if candidates[a].Readable != candidates[b].Readable {
			return candidates[a].Readable
		}
		if candidates[a].Differing != candidates[b].Differing {
			return candidates[a].Differing < candidates[b].Differing
		}
		return candidates[a].End.After(candidates[b].End)
	})
	return candidates
}

// countDifferences compares two parameter documents key by key, counting a key
// present in one and not the other as a difference.
func countDifferences(a, b map[string]any) (differing, total int) {
	keys := make(map[string]bool, len(a)+len(b))
	for k := range a {
		keys[k] = true
	}
	for k := range b {
		keys[k] = true
	}
	for k := range keys {
		total++
		left, hasLeft := a[k]
		right, hasRight := b[k]
		if hasLeft != hasRight || valueString(left) != valueString(right) {
			differing++
		}
	}
	return differing, total
}
