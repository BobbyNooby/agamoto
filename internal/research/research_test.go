package research

import (
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

func TestFormatCVEsForPrompt(t *testing.T) {
	cves := []CVE{{
		ID:          "CVE-2023-38408",
		Severity:    "HIGH",
		Score:       7.8,
		Description: "OpenSSH vulnerability",
		URL:         "https://nvd.nist.gov/vuln/detail/CVE-2023-38408",
	}}
	out := FormatCVEsForPrompt(cves)
	if len(out) < 20 {
		t.Fatalf("expected formatted CVEs, got %q", out)
	}
}

func TestFormatKEVForPrompt(t *testing.T) {
	entries := []KEVEntry{{
		CveID:                      "CVE-2023-1",
		VendorProject:              "Test",
		Product:                    "TestProduct",
		VulnerabilityName:          "Test vuln",
		KnownRansomwareCampaignUse: "Known",
		ShortDescription:           "Test description",
	}}
	out := FormatKEVForPrompt(entries)
	if len(out) < 20 {
		t.Fatalf("expected formatted KEV entries, got %q", out)
	}
}

func TestFormatKEVForPromptEmpty(t *testing.T) {
	out := FormatKEVForPrompt(nil)
	if out != "" {
		t.Fatalf("expected empty string for no KEV matches, got %q", out)
	}
}
