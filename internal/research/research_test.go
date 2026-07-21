package research

import (
	"strings"
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	c := NewCache(1 * time.Hour)
	c.Set("key", "value")
	if v, ok := c.Get("key"); !ok || v.(string) != "value" {
		t.Fatalf("expected value, got %v, ok=%v", v, ok)
	}
}

func TestCacheExpiration(t *testing.T) {
	c := NewCache(1 * time.Millisecond)
	c.Set("key", "value")
	time.Sleep(5 * time.Millisecond)
	if _, ok := c.Get("key"); ok {
		t.Fatal("expected expired entry to be missing")
	}
}

func TestExtractTextFromHTML(t *testing.T) {
	html := `<html><head><script>alert('x')</script></head><body><nav>menu</nav><article><p>Hello world</p></article></body></html>`
	text := ExtractTextFromHTML(html)
	if !strings.Contains(text, "Hello world") {
		t.Fatalf("expected article text, got %q", text)
	}
	if strings.Contains(text, "script") || strings.Contains(text, "alert") {
		t.Fatalf("expected script content removed, got %q", text)
	}
	if strings.Contains(text, "menu") {
		t.Fatalf("expected nav content removed, got %q", text)
	}
}

func TestCorpusDeduplication(t *testing.T) {
	c := NewCorpus()
	c.AddCVEs([]CVE{{ID: "CVE-2023-1"}, {ID: "CVE-2023-1"}})
	c.AddKEVs([]KEVEntry{{CveID: "CVE-2023-2"}, {CveID: "CVE-2023-2"}})
	c.AddSearchResults([]SearchResult{{URL: "https://example.com"}, {URL: "https://example.com"}})

	if len(c.CVEs) != 1 {
		t.Fatalf("expected 1 CVE, got %d", len(c.CVEs))
	}
	if len(c.KEVs) != 1 {
		t.Fatalf("expected 1 KEV, got %d", len(c.KEVs))
	}
	if len(c.WebResults) != 1 {
		t.Fatalf("expected 1 web result, got %d", len(c.WebResults))
	}
}

func TestFormatCVEsForPrompt(t *testing.T) {
	cves := []CVE{{
		ID:          "CVE-2023-38408",
		Severity:    "HIGH",
		Score:       7.8,
		Description: "OpenSSH vulnerability",
		URL:         "https://nvd.nist.gov/vuln/detail/CVE-2023-38408",
	}}
	out := FormatCVEsForPrompt(cves)
	if !strings.Contains(out, "CVE-2023-38408") {
		t.Fatalf("expected CVE ID in output, got %q", out)
	}
	if !strings.Contains(out, "HIGH") {
		t.Fatalf("expected severity in output, got %q", out)
	}
}

func TestDedupStrings(t *testing.T) {
	in := []string{"a", "b", "a", "", " c "}
	out := dedupStrings(in)
	if len(out) != 3 {
		t.Fatalf("expected 3 unique strings, got %d: %v", len(out), out)
	}
}
