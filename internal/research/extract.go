package research

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ExtractArticle fetches a URL and extracts readable article text.
func ExtractArticle(urlStr string) (string, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("fetch read: %w", err)
	}

	text := ExtractTextFromHTML(string(body))
	if len(text) > 4000 {
		text = text[:4000] + "..."
	}
	return text, nil
}

// ExtractTextFromHTML converts HTML to readable text.
func ExtractTextFromHTML(html string) string {
	// Strategy: remove script/style/nav/footer/aside elements entirely
	html = removeElement(html, "script")
	html = removeElement(html, "style")
	html = removeElement(html, "nav")
	html = removeElement(html, "footer")
	html = removeElement(html, "aside")
	html = removeElement(html, "header")
	html = removeElement(html, "form")
	html = removeElement(html, "button")
	html = removeElement(html, "iframe")
	html = removeElement(html, "noscript")

	// Try to extract main content areas first
	main := extractTagContent(html, "main")
	article := extractTagContent(html, "article")
	content := extractTagContent(html, "div")

	var best string
	if len(main) > len(best) {
		best = main
	}
	if len(article) > len(best) {
		best = article
	}
	if len(content) > len(best) {
		best = content
	}
	if best == "" {
		best = html
	}

	// Strip remaining tags
	re := regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(best, " ")

	// Collapse whitespace
	ws := regexp.MustCompile(`\s+`)
	text = ws.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

func removeElement(html, tag string) string {
	re := regexp.MustCompile(`(?i)<` + tag + `\b[^>]*>[\s\S]*?</` + tag + `>`)
	return re.ReplaceAllString(html, " ")
}

func extractTagContent(html, tag string) string {
	re := regexp.MustCompile(`(?i)<` + tag + `\b[^>]*>([\s\S]*?)</` + tag + `>`)
	matches := re.FindAllStringSubmatch(html, -1)
	var best string
	for _, m := range matches {
		if len(m) > 1 && len(m[1]) > len(best) {
			best = m[1]
		}
	}
	return best
}
