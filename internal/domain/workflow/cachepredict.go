package workflow

import (
	"sort"
	"strings"
)

// BackendKind is the execution backend a run used, narrowed to the ones whose
// call-cache behaviour Pumbaa knows how to reason about. Anything else is
// BackendUnsupported: the prediction is withheld rather than guessed, because
// the file hashing strategy — and therefore what counts as "the same input" —
// is backend and configuration dependent.
type BackendKind int

const (
	BackendUnsupported BackendKind = iota
	BackendLocal
	BackendGCP
)

func (b BackendKind) String() string {
	switch b {
	case BackendLocal:
		return "local"
	case BackendGCP:
		return "gcp"
	default:
		return "unsupported"
	}
}

// Supported reports whether cache prediction is meaningful for this backend.
func (b BackendKind) Supported() bool { return b == BackendLocal || b == BackendGCP }

// ClassifyBackend maps a Cromwell backend name to a BackendKind. Matching is
// case-insensitive and prefix-based because deployments rename backends freely
// ("PAPIv2-beta", "LocalWithDocker"); an unrecognised name is deliberately
// BackendUnsupported rather than assumed local.
func ClassifyBackend(name string) BackendKind {
	n := strings.ToLower(strings.TrimSpace(name))
	switch {
	case n == "":
		return BackendUnsupported
	case strings.HasPrefix(n, "local"):
		return BackendLocal
	case strings.HasPrefix(n, "papi"), strings.HasPrefix(n, "jes"),
		strings.HasPrefix(n, "gcpbatch"), strings.HasPrefix(n, "googlebatch"):
		return BackendGCP
	default:
		return BackendUnsupported
	}
}

// PredictedFate is what a call is expected to do on the next submission.
type PredictedFate int

const (
	// FateUnknown means no verdict could be reached — typically a missing
	// reference, an unreadable input, or an unsupported backend.
	FateUnknown PredictedFate = iota
	// FateReuse means every input to this call is unchanged, so Cromwell is
	// expected to serve it from cache.
	FateReuse
	// FateRerun means something in the call's own definition or direct inputs
	// changed. This is a root cause the user can act on.
	FateRerun
	// FateRerunDownstream means the call itself is unchanged but an upstream
	// call will rerun. This is a *probability*, not a certainty: a rerun task
	// may produce byte-identical outputs, in which case the cache still hits.
	FateRerunDownstream
)

func (f PredictedFate) String() string {
	switch f {
	case FateReuse:
		return "reuse"
	case FateRerun:
		return "rerun"
	case FateRerunDownstream:
		return "rerun (downstream)"
	default:
		return "unknown"
	}
}

// CallPrediction is the expected cache outcome for a single call.
type CallPrediction struct {
	Call string
	Fate PredictedFate
	// Reasons explains a FateRerun in the user's terms ("docker image
	// changed"), or a FateUnknown ("input not readable"). Empty for FateReuse.
	Reasons []string
	// Cause names the upstream call responsible for a FateRerunDownstream,
	// tracing to the root cause rather than the immediate parent.
	Cause string
}

// CacheForecast is the result of predicting a submission against a reference
// run. It is deliberately explicit about what it could not determine.
type CacheForecast struct {
	Reference string
	Backend   BackendKind
	Calls     []CallPrediction
	// Warnings carries every reason the forecast may be incomplete: unreadable
	// inputs, calls absent from the reference, an unsupported backend. A
	// forecast with warnings is still shown — Cromwell is the authority and
	// the user is told what was assumed.
	Warnings []string
}

// Counts tallies the forecast by fate, for the headline summary.
func (f CacheForecast) Counts() map[PredictedFate]int {
	out := make(map[PredictedFate]int, 4)
	for _, c := range f.Calls {
		out[c.Fate]++
	}
	return out
}

// RootCauses returns the calls that will rerun on their own account, which are
// the only ones the user can do anything about.
func (f CacheForecast) RootCauses() []CallPrediction {
	var out []CallPrediction
	for _, c := range f.Calls {
		if c.Fate == FateRerun {
			out = append(out, c)
		}
	}
	return out
}

// CallAssessment is what was determined about one call on its own account,
// before the dependency graph is taken into account. The zero value means
// "nothing changed and everything was verified", which is what makes a call
// eligible for reuse.
type CallAssessment struct {
	// Reasons, when non-empty, are why the call will rerun on its own account.
	// This is a root cause: something in its own fingerprint changed.
	Reasons []string
	// Unknown, when non-empty, means nothing could be established about the
	// call at all. It poisons descendants, since a call downstream of an
	// unknowable one is itself unknowable.
	Unknown string
	// ReuseBlocked, when non-empty, means no change was found but the call
	// cannot be shown unchanged either: some input resisted verification. Such
	// a call may still inherit a rerun from upstream — which is more
	// informative than Unknown — but it can never be reported as reuse.
	//
	// The distinction matters because these two are not the same claim:
	// Unknown says "I know nothing"; ReuseBlocked says "I found nothing wrong
	// and could not finish checking".
	ReuseBlocked string
}

// PredictCacheReuse propagates per-call assessments through the workflow's
// dependency graph to a fate for every call.
//
// graph maps a call to the calls it consumes outputs from. Calls with no
// assessment are taken as verified-unchanged, which is the whole point: they
// are the ones that will be reused.
func PredictCacheReuse(graph map[string][]string, assessments map[string]CallAssessment) []CallPrediction {
	names := make([]string, 0, len(graph))
	for name := range graph {
		names = append(names, name)
	}
	sort.Strings(names)

	memo := make(map[string]CallPrediction, len(graph))
	// visiting guards against a cyclic graph, which a valid WDL cannot produce
	// but a partially-parsed one might.
	visiting := make(map[string]bool, len(graph))

	var resolve func(name string) CallPrediction
	resolve = func(name string) CallPrediction {
		if p, ok := memo[name]; ok {
			return p
		}
		if visiting[name] {
			return CallPrediction{Call: name, Fate: FateUnknown, Reasons: []string{"cyclic dependency"}}
		}
		visiting[name] = true
		defer func() { visiting[name] = false }()

		assessment := assessments[name]
		p := CallPrediction{Call: name}
		switch {
		case assessment.Unknown != "":
			p.Fate = FateUnknown
			p.Reasons = []string{assessment.Unknown}
		case len(assessment.Reasons) > 0:
			p.Fate = FateRerun
			p.Reasons = assessment.Reasons
		default:
			p.Fate = FateReuse
			// An upstream verdict overrides reuse: unknown wins over rerun,
			// because an unknowable input makes the whole subtree unknowable.
			for _, up := range graph[name] {
				u := resolve(up)
				switch u.Fate {
				case FateUnknown:
					p.Fate = FateUnknown
					p.Cause = rootCause(u, up)
					p.Reasons = []string{"upstream fate unknown"}
				case FateRerun, FateRerunDownstream:
					if p.Fate != FateUnknown {
						p.Fate = FateRerunDownstream
						p.Cause = rootCause(u, up)
					}
				}
			}
			// Nothing upstream forced a rerun, so reuse would be the verdict —
			// but it is only available to a call whose inputs were fully
			// verified. Otherwise the honest answer is that we do not know.
			if p.Fate == FateReuse && assessment.ReuseBlocked != "" {
				p.Fate = FateUnknown
				p.Reasons = []string{assessment.ReuseBlocked}
			}
		}

		memo[name] = p
		return p
	}

	out := make([]CallPrediction, 0, len(names))
	for _, name := range names {
		out = append(out, resolve(name))
	}
	return out
}

// rootCause reports the call ultimately responsible for an upstream verdict,
// so a chain A→B→C blames A rather than naming each hop.
func rootCause(upstream CallPrediction, upstreamName string) string {
	if upstream.Cause != "" {
		return upstream.Cause
	}
	return upstreamName
}
