package workflow

import (
	"sort"
	"strings"
)

// CallFingerprint is a call's call-caching hash map in flattened form: the
// keys Cromwell nests under groups ("runtime attribute" → "docker") become
// single "runtime attribute: docker" keys, so two calls can be compared with
// a plain map diff.
//
// Cromwell decides a cache hit by comparing exactly these values, so two calls
// with equal fingerprints are interchangeable as far as the cache is concerned.
type CallFingerprint map[string]string

// FlattenHashes converts the nested `callCaching.hashes` tree from Cromwell
// metadata into a CallFingerprint. Non-string leaves are skipped: every hash
// Cromwell emits is a hex string, and anything else is a payload we do not
// understand and must not silently compare.
func FlattenHashes(tree map[string]any) CallFingerprint {
	out := make(CallFingerprint)
	flattenHashTree(tree, "", out)
	return out
}

func flattenHashTree(node map[string]any, prefix string, out CallFingerprint) {
	for k, v := range node {
		key := prefix + k
		switch val := v.(type) {
		case map[string]any:
			flattenHashTree(val, key+": ", out)
		case string:
			out[key] = val
		}
	}
}

// HashChange is a single differing entry between two fingerprints. An empty
// Reference means the key only exists in the current call; an empty Current
// means it only exists in the reference.
type HashChange struct {
	Key       string
	Reference string
	Current   string
	Category  ChangeCategory
}

// ChangeCategory groups hash keys by what the user would have to change to
// cause them, so a report can talk about "docker" instead of echoing two
// unrelated-looking hash keys.
type ChangeCategory int

const (
	// CategoryOther covers keys this version does not recognise. Reports must
	// render it by echoing the raw key rather than inventing an explanation,
	// since the hash taxonomy is Cromwell-version dependent.
	CategoryOther ChangeCategory = iota
	CategoryDocker
	CategoryCommand
	CategoryInputFile
	CategoryInputValue
	CategoryRuntime
	CategoryBackend
	CategoryCount
)

func (c ChangeCategory) String() string {
	switch c {
	case CategoryDocker:
		return "docker image"
	case CategoryCommand:
		return "command template"
	case CategoryInputFile:
		return "input file"
	case CategoryInputValue:
		return "input value"
	case CategoryRuntime:
		return "runtime attribute"
	case CategoryBackend:
		return "backend"
	case CategoryCount:
		return "input/output count"
	default:
		return "other"
	}
}

const (
	inputPrefix   = "input: "
	runtimePrefix = "runtime attribute: "
	outputPrefix  = "output expression: "
)

// categorize maps a flattened hash key to the change it represents.
//
// Note that a docker change surfaces under two keys when the WDL passes docker
// as a task input ("runtime attribute: docker" and "input: String docker").
// Both map to CategoryDocker so a report can collapse them into one finding.
func categorize(key string) ChangeCategory {
	switch {
	case key == "command template":
		return CategoryCommand
	case key == "backend name":
		return CategoryBackend
	case key == "input count" || key == "output count":
		return CategoryCount
	case key == runtimePrefix+"docker":
		return CategoryDocker
	case strings.HasPrefix(key, runtimePrefix):
		return CategoryRuntime
	case strings.HasPrefix(key, inputPrefix):
		typ, name := ParseInputHashKey(key)
		if name == "docker" {
			return CategoryDocker
		}
		if typ == "File" {
			return CategoryInputFile
		}
		return CategoryInputValue
	case strings.HasPrefix(key, outputPrefix):
		return CategoryOther
	default:
		return CategoryOther
	}
}

// ParseInputHashKey splits an "input: <Type> <name>" hash key into its declared
// type and input name. It returns empty strings for any key that is not an
// input entry or does not carry both parts.
func ParseInputHashKey(key string) (declaredType, name string) {
	rest, ok := strings.CutPrefix(key, inputPrefix)
	if !ok {
		return "", ""
	}
	// Cromwell renders compound types with spaces ("Array[File] xs"), so the
	// name is the last field and the type is everything before it.
	idx := strings.LastIndex(rest, " ")
	if idx <= 0 {
		return "", ""
	}
	return rest[:idx], rest[idx+1:]
}

// CompareFingerprints reports every key whose hash differs between a reference
// call and the current one, sorted by key so output is stable. An empty result
// means the two calls are cache-equivalent.
func CompareFingerprints(reference, current CallFingerprint) []HashChange {
	seen := make(map[string]struct{}, len(reference)+len(current))
	for k := range reference {
		seen[k] = struct{}{}
	}
	for k := range current {
		seen[k] = struct{}{}
	}

	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var changes []HashChange
	for _, k := range keys {
		ref, cur := reference[k], current[k]
		if ref == cur {
			continue
		}
		changes = append(changes, HashChange{
			Key:       k,
			Reference: ref,
			Current:   cur,
			Category:  categorize(k),
		})
	}
	return changes
}

// Categories returns the distinct categories present in a set of changes, in
// declaration order, so a report can say "docker image, command template"
// instead of listing every raw key.
func Categories(changes []HashChange) []ChangeCategory {
	var out []ChangeCategory
	seen := make(map[ChangeCategory]bool, len(changes))
	for c := CategoryOther; c <= CategoryCount; c++ {
		for _, ch := range changes {
			if ch.Category == c && !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	return out
}
