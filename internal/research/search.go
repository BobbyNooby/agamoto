package research

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const ddgLiteURL = "https://lite.duckduckgo.com/lite"

// SearchResult is a single web search result.
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

// SearchDuckDuckGo queries DuckDuckGo Lite and returns parsed results.
func SearchDuckDuckGo(query string) ([]SearchResult, error) {
	u, _ := url.Parse(ddgLiteURL)
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ddg search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ddg: %s: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ddg read: %w", err)
	}

	return parseDDGResults(string(body)), nil
}

func parseDDGResults(html string) []SearchResult {
	var results []SearchResult

	// DuckDuckGo Lite uses a table with result rows.
	// Each result has a link in a <a> tag, usually preceded by a number in a <td>.
	// Links look like: <a rel="nofollow" href="...">Title</a>
	linkRe := regexp.MustCompile(`<a[^>]+href="([^"]+)"[^>]*>([^<]+)</a>`)
	// Snippets are usually in the next <td> after the link.
	tdRe := regexp.MustCompile(`<td[^>]*>(.*?)</td>`)

	matches := linkRe.FindAllStringSubmatch(html, -1)
	tds := tdRe.FindAllStringSubmatch(html, -1)

	seen := make(map[string]bool)
	snippetIdx := 0

	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		rawURL := m[1]
		title := strings.TrimSpace(stripHTML(m[2]))

		// Skip pagination / navigation links
		if strings.Contains(rawURL, "duckduckgo.com") || strings.Contains(rawURL, "javascript:") {
			continue
		}

		// Resolve DDG redirect wrappers
		if strings.HasPrefix(rawURL, "//") {
			rawURL = "https:" + rawURL
		}
		if strings.HasPrefix(rawURL, "/") {
			rawURL = "https://lite.duckduckgo.com" + rawURL
		}

		// DDG sometimes wraps URLs in a redirect
		if strings.Contains(rawURL, "duckduckgo.com/l/?") {
			u, _ := url.Parse(rawURL)
			if u != nil {
				if target := u.Query().Get("uddg"); target != "" {
					rawURL = target
				}
			}
		}

		if rawURL == "" || seen[rawURL] {
			continue
		}
		seen[rawURL] = true

		// Find a snippet near this link
		snippet := ""
		for ; snippetIdx < len(tds); snippetIdx++ {
			text := strings.TrimSpace(stripHTML(tds[snippetIdx][1]))
			if text != "" && text != title && !strings.Contains(text, "Next") && !strings.Contains(text, "Previous") {
				snippet = text
				snippetIdx++
				break
			}
		}

		results = append(results, SearchResult{
			Title:   title,
			URL:     rawURL,
			Snippet: snippet,
		})
	}

	return results
}

func stripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return re.ReplaceAllString(s, "")
}

// FormatSearchResultsForPrompt formats search snippets for the AI prompt.
func FormatSearchResultsForPrompt(results []SearchResult) string {
	if len(results) == 0 {
		return "No web search results found."
	}
	var b strings.Builder
	b.WriteString("### Web Search Results\n")
	for _, r := range results {
		b.WriteString(fmt.Sprintf("- **%s** (%s)\n  %s\n", r.Title, r.URL, r.Snippet))
	}
	return b.String()
}
