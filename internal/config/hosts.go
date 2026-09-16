package config

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// DefaultCromwellHost is used when no host was configured anywhere.
const DefaultCromwellHost = "http://localhost:8000"

// Host aliases exist so a Cromwell URL is typed once, at registration, and
// referred to by name everywhere else (`--host prod`). A reference is
// resolved against the registry first, so a registered alias always wins;
// only an unregistered reference is guessed at, and the rule for that guess
// is deliberately blunt: aliases are bare words, so anything carrying a
// scheme, a dot, a colon or a slash is a URL.
var aliasPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// HostRef is a resolved host: its URL, plus the alias it came from when it
// came from one. The alias is worth keeping because it is what the user
// recognises — in prompts, in the dashboard header, and in the run history.
type HostRef struct {
	Alias string
	URL   string
}

// Display returns the alias when there is one, falling back to the URL.
func (h HostRef) Display() string {
	if h.Alias != "" {
		return h.Alias
	}
	return h.URL
}

// UnknownHostError reports a reference that is neither a registered alias nor
// something that could be a URL. It carries the registry so the message can
// show the way out instead of just the dead end.
type UnknownHostError struct {
	Ref   string
	Known []string
}

func (e *UnknownHostError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "unknown host %q", e.Ref)
	if suggestion := closestAlias(e.Ref, e.Known); suggestion != "" {
		fmt.Fprintf(&b, " (did you mean %q?)", suggestion)
	}
	if len(e.Known) > 0 {
		fmt.Fprintf(&b, "; registered hosts: %s", strings.Join(e.Known, ", "))
	} else {
		b.WriteString("; no hosts registered yet")
	}
	fmt.Fprintf(&b, ". Register it with `pumbaa host add %s <url>`, or pass a full URL", e.Ref)
	return b.String()
}

// ValidateAlias rejects names that could be mistaken for a URL, which would
// make the reference ambiguous at resolution time.
func ValidateAlias(alias string) error {
	if alias == "" {
		return fmt.Errorf("host alias cannot be empty")
	}
	if !aliasPattern.MatchString(alias) {
		return fmt.Errorf("invalid host alias %q: use letters, digits, %q and %q, starting with a letter or digit", alias, "-", "_")
	}
	return nil
}

// looksLikeURL is the guess applied to references the registry does not know.
func looksLikeURL(ref string) bool {
	return strings.ContainsAny(ref, ":./") || !aliasPattern.MatchString(ref)
}

// NormalizeHostURL makes a URL canonical enough to be used as an identity:
// same server, same string. It fills in the scheme most Cromwell servers are
// reached by and drops the trailing slash, so `localhost:8000` and
// `http://localhost:8000/` are one host rather than three.
func NormalizeHostURL(raw string) string {
	url := strings.TrimSpace(raw)
	if url == "" {
		return ""
	}
	if !strings.Contains(url, "://") {
		url = "http://" + url
	}
	return strings.TrimRight(url, "/")
}

// ResolveHost turns a host reference into a URL. An empty reference yields
// the configured default.
func (c *FileConfig) ResolveHost(ref string) (HostRef, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return c.defaultHost(), nil
	}
	if url, ok := c.Hosts[ref]; ok {
		return HostRef{Alias: ref, URL: NormalizeHostURL(url)}, nil
	}
	if looksLikeURL(ref) {
		return HostRef{URL: NormalizeHostURL(ref)}, nil
	}
	return HostRef{}, &UnknownHostError{Ref: ref, Known: c.HostAliases()}
}

// defaultHost applies the precedence for "no host was asked for": the alias
// marked as default, then the single-host `cromwell_host` setting that
// predates the registry, then the built-in default.
func (c *FileConfig) defaultHost() HostRef {
	if c.DefaultHost != "" {
		if url, ok := c.Hosts[c.DefaultHost]; ok {
			return HostRef{Alias: c.DefaultHost, URL: NormalizeHostURL(url)}
		}
	}
	if c.CromwellHost != "" {
		return HostRef{URL: NormalizeHostURL(c.CromwellHost)}
	}
	return HostRef{URL: DefaultCromwellHost}
}

// HostAliases returns the registered aliases in a stable order.
func (c *FileConfig) HostAliases() []string {
	aliases := make([]string, 0, len(c.Hosts))
	for alias := range c.Hosts {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

// AddHost registers (or replaces) an alias. The first host registered becomes
// the default: a registry with one host and no default would be a trap.
func (c *FileConfig) AddHost(alias, url string) error {
	if err := ValidateAlias(alias); err != nil {
		return err
	}
	normalized := NormalizeHostURL(url)
	if normalized == "" {
		return fmt.Errorf("host URL cannot be empty")
	}
	if c.Hosts == nil {
		c.Hosts = make(map[string]string)
	}
	firstHost := len(c.Hosts) == 0
	c.Hosts[alias] = normalized
	if firstHost || c.DefaultHost == "" {
		c.DefaultHost = alias
	}
	return nil
}

// RemoveHost drops an alias, moving the default elsewhere when the removed
// host was the default.
func (c *FileConfig) RemoveHost(alias string) error {
	if _, ok := c.Hosts[alias]; !ok {
		return &UnknownHostError{Ref: alias, Known: c.HostAliases()}
	}
	delete(c.Hosts, alias)
	if c.DefaultHost == alias {
		c.DefaultHost = ""
		if remaining := c.HostAliases(); len(remaining) > 0 {
			c.DefaultHost = remaining[0]
		}
	}
	return nil
}

// SetDefaultHost marks a registered alias as the one used when no host is
// given.
func (c *FileConfig) SetDefaultHost(alias string) error {
	if _, ok := c.Hosts[alias]; !ok {
		return &UnknownHostError{Ref: alias, Known: c.HostAliases()}
	}
	c.DefaultHost = alias
	return nil
}

// closestAlias returns the registered alias within a small edit distance of
// the reference, which catches the typo without inventing a match for a name
// the user never registered.
func closestAlias(ref string, known []string) string {
	best, bestDistance := "", 3
	for _, alias := range known {
		d := editDistance(strings.ToLower(ref), strings.ToLower(alias))
		if d < bestDistance {
			best, bestDistance = alias, d
		}
	}
	return best
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = minOf(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func minOf(values ...int) int {
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}
