package research

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const cisaKEVURL = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"

// KEVEntry is a single entry in the CISA KEV catalog.
type KEVEntry struct {
	CveID                      string   `json:"cveID"`
	VendorProject              string   `json:"vendorProject"`
	Product                    string   `json:"product"`
	VulnerabilityName          string   `json:"vulnerabilityName"`
	DateAdded                  string   `json:"dateAdded"`
	ShortDescription           string   `json:"shortDescription"`
	RequiredAction             string   `json:"requiredAction"`
	DueDate                    string   `json:"dueDate"`
	KnownRansomwareCampaignUse string   `json:"knownRansomwareCampaignUse"`
	Notes                      string   `json:"notes"`
	CWEs                       []string `json:"cwes"`
}

// KEVCatalog holds the full CISA KEV dataset.
type KEVCatalog struct {
	Title           string     `json:"title"`
	CatalogVersion  string     `json:"catalogVersion"`
	DateReleased    string     `json:"dateReleased"`
	Count           int        `json:"count"`
	Vulnerabilities []KEVEntry `json:"vulnerabilities"`
}

// KEVLoader fetches and caches the CISA KEV catalog.
type KEVLoader struct {
	HTTP  *http.Client
	cache *Cache
}

func NewKEVLoader() *KEVLoader {
	return &KEVLoader{
		HTTP:  &http.Client{Timeout: 30 * time.Second},
		cache: NewCache(24 * time.Hour),
	}
}

func kevCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "agamoto", "cisa_kev.json"), nil
}

// Load fetches the KEV catalog from CISA or local cache.
func (k *KEVLoader) Load() (*KEVCatalog, error) {
	cacheKey := "cisa_kev:full"
	if v, ok := k.cache.Get(cacheKey); ok {
		return v.(*KEVCatalog), nil
	}

	// Try local file cache first
	path, err := kevCachePath()
	if err == nil {
		if data, err := os.ReadFile(path); err == nil {
			var catalog KEVCatalog
			if err := json.Unmarshal(data, &catalog); err == nil {
				// Check if file is fresh (< 24h)
				info, err := os.Stat(path)
				if err == nil && time.Since(info.ModTime()) < 24*time.Hour {
					k.cache.Set(cacheKey, &catalog)
					return &catalog, nil
				}
			}
		}
	}

	// Fetch from CISA
	resp, err := k.HTTP.Get(cisaKEVURL)
	if err != nil {
		return nil, fmt.Errorf("kev fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("kev: %s: %s", resp.Status, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("kev read: %w", err)
	}

	var catalog KEVCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("kev decode: %w", err)
	}

	// Write to local cache
	if path != "" {
		_ = os.MkdirAll(filepath.Dir(path), 0755)
		_ = os.WriteFile(path, data, 0644)
	}

	k.cache.Set(cacheKey, &catalog)
	return &catalog, nil
}

// FindByCVE checks if a specific CVE ID is in the KEV catalog.
func (c *KEVCatalog) FindByCVE(cveID string) *KEVEntry {
	for i := range c.Vulnerabilities {
		if strings.EqualFold(c.Vulnerabilities[i].CveID, cveID) {
			return &c.Vulnerabilities[i]
		}
	}
	return nil
}

// FindByProduct does fuzzy matching on vendor/project + product fields.
func (c *KEVCatalog) FindByProduct(vendor, product string) []KEVEntry {
	var matches []KEVEntry
	for _, e := range c.Vulnerabilities {
		if product != "" && strings.Contains(strings.ToLower(e.Product), strings.ToLower(product)) {
			matches = append(matches, e)
			continue
		}
		if vendor != "" && strings.Contains(strings.ToLower(e.VendorProject), strings.ToLower(vendor)) {
			matches = append(matches, e)
		}
	}
	return matches
}

// FormatKEVForPrompt formats KEV entries for the AI prompt.
func FormatKEVForPrompt(entries []KEVEntry) string {
	if len(entries) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("### Actively Exploited Vulnerabilities (CISA KEV)\n")
	for _, e := range entries {
		b.WriteString(fmt.Sprintf("- **%s** | %s | %s\n  %s\n  Ransomware use: %s | Due: %s\n", e.CveID, e.VendorProject, e.Product, e.ShortDescription, e.KnownRansomwareCampaignUse, e.DueDate))
	}
	return b.String()
}
