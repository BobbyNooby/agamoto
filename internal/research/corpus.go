package research

import (
	"fmt"
	"strings"
)

// ResearchCorpus holds all gathered intelligence for a target.
type ResearchCorpus struct {
	CVEs           []CVE
	KEVs           []KEVEntry
	WebResults     []SearchResult
	Articles       []ExtractedArticle
}

// ExtractedArticle is a fetched web page with extracted text.
type ExtractedArticle struct {
	Title   string
	URL     string
	Content string
}

// NewCorpus creates an empty corpus.
func NewCorpus() *ResearchCorpus {
	return &ResearchCorpus{}
}

// AddCVEs adds CVEs, deduplicating by ID.
func (c *ResearchCorpus) AddCVEs(cves []CVE) {
	seen := make(map[string]bool)
	for _, cve := range c.CVEs {
		seen[cve.ID] = true
	}
	for _, cve := range cves {
		if !seen[cve.ID] {
			seen[cve.ID] = true
			c.CVEs = append(c.CVEs, cve)
		}
	}
}

// AddKEVs adds KEV entries, deduplicating by CVE ID.
func (c *ResearchCorpus) AddKEVs(entries []KEVEntry) {
	seen := make(map[string]bool)
	for _, e := range c.KEVs {
		seen[e.CveID] = true
	}
	for _, e := range entries {
		if !seen[e.CveID] {
			seen[e.CveID] = true
			c.KEVs = append(c.KEVs, e)
		}
	}
}

// AddSearchResults adds search results, deduplicating by URL.
func (c *ResearchCorpus) AddSearchResults(results []SearchResult) {
	seen := make(map[string]bool)
	for _, r := range c.WebResults {
		seen[r.URL] = true
	}
	for _, r := range results {
		if !seen[r.URL] {
			seen[r.URL] = true
			c.WebResults = append(c.WebResults, r)
		}
	}
}

// AddArticle adds an extracted article, deduplicating by URL.
func (c *ResearchCorpus) AddArticle(a ExtractedArticle) {
	for _, existing := range c.Articles {
		if existing.URL == a.URL {
			return
		}
	}
	c.Articles = append(c.Articles, a)
}

// Format returns a markdown string suitable for the AI prompt.
func (c *ResearchCorpus) Format() string {
	var b strings.Builder

	b.WriteString("## CVE Intelligence\n")
	b.WriteString(FormatCVEsForPrompt(c.CVEs))
	b.WriteString("\n")

	if len(c.KEVs) > 0 {
		b.WriteString(FormatKEVForPrompt(c.KEVs))
		b.WriteString("\n")
	}

	if len(c.Articles) > 0 {
		b.WriteString("### Extracted Web Articles\n")
		for _, a := range c.Articles {
			b.WriteString(fmt.Sprintf("- **%s** (%s)\n  %s\n", a.Title, a.URL, truncate(a.Content, 600)))
		}
		b.WriteString("\n")
	}

	b.WriteString(FormatSearchResultsForPrompt(c.WebResults))

	return strings.TrimSpace(b.String())
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
