package research

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DeepResearchOptions controls the iterative research loop.
type DeepResearchOptions struct {
	MaxPasses       int
	MaxURLsPerQuery int
	Target          string
	Services        []ServiceFingerprint
}

// RunDeepResearch executes an iterative DDG + extraction research loop.
func RunDeepResearch(opts DeepResearchOptions) (*ResearchCorpus, error) {
	if opts.MaxPasses <= 0 {
		opts.MaxPasses = 1
	}
	if opts.MaxURLsPerQuery <= 0 {
		opts.MaxURLsPerQuery = 5
	}

	corpus := NewCorpus()
	queries := buildSeedQueries(opts.Target, opts.Services)
	seenQueries := make(map[string]bool)
	seenURLs := make(map[string]bool)

	for pass := 1; pass <= opts.MaxPasses; pass++ {
		newQueries := []string{}

		for _, q := range queries {
			if seenQueries[q] {
				continue
			}
			seenQueries[q] = true

			results, err := SearchDuckDuckGo(q)
			if err != nil {
				continue
			}

			// Limit results per query
			if len(results) > opts.MaxURLsPerQuery {
				results = results[:opts.MaxURLsPerQuery]
			}

			corpus.AddSearchResults(results)

			// In deep mode, fetch full article text
			for _, r := range results {
				if seenURLs[r.URL] {
					continue
				}
				seenURLs[r.URL] = true

				text, err := ExtractArticle(r.URL)
				if err != nil || len(text) < 100 {
					continue
				}

				corpus.AddArticle(ExtractedArticle{
					Title:   r.Title,
					URL:     r.URL,
					Content: text,
				})
			}

			// Generate follow-up queries from content (heuristic)
			if pass < opts.MaxPasses {
				newQueries = append(newQueries, generateFollowUps(q, results, corpus)...)
			}
		}

		queries = dedupStrings(newQueries)
		if len(queries) == 0 {
			break
		}

		// Be polite to DDG
		time.Sleep(1 * time.Second)
	}

	return corpus, nil
}

// BasicResearch runs a single-pass DDG search without full extraction.
func BasicResearch(target string, services []ServiceFingerprint) (*ResearchCorpus, error) {
	corpus := NewCorpus()
	queries := buildSeedQueries(target, services)
	seenQueries := make(map[string]bool)

	for _, q := range queries {
		if seenQueries[q] {
			continue
		}
		seenQueries[q] = true

		results, err := SearchDuckDuckGo(q)
		if err != nil {
			continue
		}
		// Only top 3 snippets
		if len(results) > 3 {
			results = results[:3]
		}
		corpus.AddSearchResults(results)
	}

	return corpus, nil
}

func buildSeedQueries(target string, services []ServiceFingerprint) []string {
	queries := []string{
		fmt.Sprintf("%s vulnerabilities", target),
		fmt.Sprintf("%s CVE", target),
	}

	for _, svc := range services {
		if svc.Product == "" {
			continue
		}
		q := svc.Product
		if svc.Version != "" {
			q += " " + svc.Version
		}
		queries = append(queries, fmt.Sprintf("%s vulnerability", q))
		queries = append(queries, fmt.Sprintf("%s CVE", q))
	}

	return dedupStrings(queries)
}

func generateFollowUps(parentQuery string, results []SearchResult, corpus *ResearchCorpus) []string {
	var followUps []string

	// Follow up on CVEs found in NVD
	for _, cve := range corpus.CVEs {
		followUps = append(followUps, fmt.Sprintf("%s exploit POC github", cve.ID))
		followUps = append(followUps, fmt.Sprintf("%s proof of concept", cve.ID))
	}

	// Follow up on target breach/discussions
	tokens := strings.Fields(parentQuery)
	if len(tokens) > 0 {
		base := tokens[0]
		followUps = append(followUps, fmt.Sprintf("%s breach %d", base, time.Now().Year()))
		followUps = append(followUps, fmt.Sprintf("%s security advisory", base))
	}

	return dedupStrings(followUps)
}

func dedupStrings(in []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func parseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
