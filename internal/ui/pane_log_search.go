package ui

import (
	"fmt"
	"strings"

	"github.com/eurico-martins/lazytree/internal/git"

	"github.com/charmbracelet/lipgloss"
	"github.com/eurico-martins/lazytree/internal/model"
)

// logSearch is the state of a confirmed or in-progress `/` search. Unlike a
// filter it hides nothing: every commit stays visible, matches are flagged
// and highlighted, and the cursor jumps between them (n/N).
type logSearch struct {
	query   string // lowercased; "" means no active search
	isMatch []bool // per commit index, same length as the commit list
	count   int    // number of true entries in isMatch

	// A deep search (g:/s:/p: prefix) is answered by git over the whole
	// history: hashes holds every matching commit, and label is what the
	// footer shows. query then carries the label (non-empty marks the
	// search active) but nothing is highlighted in the text.
	deep   bool
	hashes map[string]bool
	label  string
}

// highlightQuery is the substring to mark in row text: the query for an
// in-memory search, nothing for a deep one (git decided what matched).
func (s *logSearch) highlightQuery() string {
	if s.deep {
		return ""
	}
	return s.query
}

// matchesCommit reports whether c is a hit under this search.
func (s *logSearch) matchesCommit(c model.Commit) bool {
	if s.deep {
		return s.hashes[c.Hash]
	}
	return commitMatches(c, s.query)
}

func (s *logSearch) matches(commitIdx int) bool {
	return s.query != "" && commitIdx >= 0 && commitIdx < len(s.isMatch) && s.isMatch[commitIdx]
}

// commitMatches reports whether the lowercased query q occurs in any
// searchable field of c: subject, body, hashes, author, or a ref name.
func commitMatches(c model.Commit, q string) bool {
	if strings.Contains(strings.ToLower(c.Subject), q) ||
		strings.Contains(strings.ToLower(c.Body), q) ||
		strings.Contains(c.Hash, q) ||
		strings.Contains(strings.ToLower(c.Author), q) {
		return true
	}
	for _, r := range c.Refs {
		if strings.Contains(strings.ToLower(r.Name), q) {
			return true
		}
	}
	return false
}

// setSearch starts (or replaces) the search and rescans every commit. An
// empty query clears it. It does not move the cursor.
func (p *logPane) setSearch(query string) {
	p.search = logSearch{query: strings.ToLower(strings.TrimSpace(query))}
	p.extendSearch(0)
}

// setDeepSearch installs the result of a git-side search: hashes are the
// matching commits across the whole history, label names the search.
func (p *logPane) setDeepSearch(label string, hashes []string) {
	set := make(map[string]bool, len(hashes))
	for _, h := range hashes {
		set[h] = true
	}
	p.search = logSearch{query: strings.ToLower(label), deep: true, hashes: set, label: label}
	p.extendSearch(0)
}

func (p *logPane) clearSearch() {
	p.search = logSearch{}
}

func (p *logPane) searchActive() bool {
	return p.search.query != ""
}

// extendSearch scans commits from index `from` on, keeping isMatch in step
// with the commit list as pages are appended (from == 0 rescans all).
func (p *logPane) extendSearch(from int) {
	s := &p.search
	if s.query == "" {
		return
	}
	from = min(max(from, 0), len(s.isMatch))
	for _, m := range s.isMatch[from:] {
		if m {
			s.count--
		}
	}
	s.isMatch = s.isMatch[:from]
	for i := from; i < len(p.commits); i++ {
		m := s.matchesCommit(p.commits[i])
		s.isMatch = append(s.isMatch, m)
		if m {
			s.count++
		}
	}
}

// jumpToMatch moves the cursor to the next (or previous) match, wrapping
// around the list. With inclusive set, the commit under the cursor counts
// as the first candidate. It reports whether a match was found.
func (p *logPane) jumpToMatch(forward, inclusive bool) bool {
	n := len(p.commits)
	if !p.searchActive() || n == 0 || p.search.count == 0 {
		return false
	}
	step := 1
	if !forward {
		step = -1
	}
	start := p.cursor
	if !inclusive {
		start += step
	}
	for i := range n {
		idx := ((start+step*i)%n + n) % n
		if idx < len(p.search.isMatch) && p.search.isMatch[idx] {
			p.cursor = idx
			p.ensureVisible()
			return true
		}
	}
	return false
}

// setCursor restores a cursor/scroll position (clamped to the list).
func (p *logPane) setCursor(cursor, offset int) {
	p.cursor, p.offset = cursor, offset
	p.ensureVisible()
}

// searchStatus summarizes the active search for the footer: the query and
// the cursor's position among the matches, e.g. "/fix 2/5".
func (p *logPane) searchStatus() string {
	if !p.searchActive() {
		return ""
	}
	if p.search.count == 0 {
		return fmt.Sprintf("/%s: no matches", p.search.display())
	}
	pos := 0
	for i := 0; i <= p.cursor && i < len(p.search.isMatch); i++ {
		if p.search.isMatch[i] {
			pos++
		}
	}
	return fmt.Sprintf("/%s %d/%d", p.search.display(), pos, p.search.count)
}

func (s *logSearch) display() string {
	if s.deep {
		return s.label
	}
	return s.query
}

// parseDeepQuery recognizes the git-side search prefixes: "g:" (commit
// message, --grep), "s:" (code added/removed, -S) and "p:" (path). ok is
// false for an ordinary in-memory query or an empty term.
func parseDeepQuery(input string) (kind git.SearchKind, term string, ok bool) {
	input = strings.TrimSpace(input)
	if len(input) < 3 || input[1] != ':' {
		return 0, "", false
	}
	switch input[0] {
	case 'g':
		kind = git.SearchGrep
	case 's':
		kind = git.SearchCode
	case 'p':
		kind = git.SearchPath
	default:
		return 0, "", false
	}
	term = strings.TrimSpace(input[2:])
	return kind, term, term != ""
}

// highlightMatches renders text in base style, with each occurrence of the
// lowercased query lq drawn in styleMatch. text must be plain (no ANSI).
func highlightMatches(text, lq string, base lipgloss.Style) string {
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	// Lowercasing can change byte lengths for some runes; then the byte
	// offsets found in `lower` don't map back onto text, so skip marking.
	if lq == "" || len(lower) != len(text) {
		return base.Render(text)
	}
	var sb strings.Builder
	pos := 0
	for {
		i := strings.Index(lower[pos:], lq)
		if i < 0 {
			break
		}
		start := pos + i
		if start > pos {
			sb.WriteString(base.Render(text[pos:start]))
		}
		sb.WriteString(styleMatch.Render(text[start : start+len(lq)]))
		pos = start + len(lq)
	}
	if pos < len(text) {
		sb.WriteString(base.Render(text[pos:]))
	}
	return sb.String()
}
