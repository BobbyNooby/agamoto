package research

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const nvdAPIBase = "https://services.nvd.nist.gov/rest/json/cves/2.0"

// NVDClient queries the National Vulnerability Database.
type NVDClient struct {
	APIKey string       // optional; higher rate limits if set
	HTTP   *http.Client
	cache  *Cache
}

func NewNVDClient(apiKey string) *NVDClient {
	return &NVDClient{
		APIKey: apiKey,
		HTTP:   &http.Client{Timeout: 10 * time.Second},
		cache:  NewCache(1 * time.Hour),
	}
}

type nvdResponse struct {
	ResultsPerPage  int        `json:"resultsPerPage"`
	Vulnerabilities []nvdVuln  `json:"vulnerabilities"`
}

type nvdVuln struct {
	CVE nvdCVE `json:"cve"`
}

type nvdCVE struct {
	ID           string    `json:"id"`
	Descriptions []nvdDesc `json:"descriptions"`
	Metrics      nvdMetrics `json:"metrics"`
}

type nvdDesc struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

type nvdMetrics struct {
	CVSSMetricV31 []nvdCVSS31 `json:"cvssMetricV31"`
	CVSSMetricV30 []nvdCVSS30 `json:"cvssMetricV30"`
	CVSSMetricV2  []nvdCVSS2  `json:"cvssMetricV2"`
}

type nvdCVSS31 struct {
	CVSSData nvdCVSSData `json:"cvssData"`
}
type nvdCVSS30 struct {
	CVSSData nvdCVSSData `json:"cvssData"`
}
type nvdCVSS2 struct {
	CVSSData nvdCVSSDataV2 `json:"cvssData"`
}

type nvdCVSSData struct {
	BaseScore    float64 `json:"baseScore"`
	BaseSeverity string  `json:"baseSeverity"`
}
type nvdCVSSDataV2 struct {
	BaseScore float64 `json:"baseScore"`
}

// CVE represents a single vulnerability finding.
type CVE struct {
	ID          string  `json:"id"`
	Severity    string  `json:"severity"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
	URL         string  `json:"url"`
}

// Query searches NVD for CVEs matching a keyword (typically "product version").
func (c *NVDClient) Query(keyword string) ([]CVE, error) {
	cacheKey := "nvd:" + keyword
	if v, ok := c.cache.Get(cacheKey); ok {
		return v.([]CVE), nil
	}

	u, _ := url.Parse(nvdAPIBase)
	q := u.Query()
	q.Set("keywordSearch", keyword)
	q.Set("resultsPerPage", "5")
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	if c.APIKey != "" {
		req.Header.Set("apiKey", c.APIKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nvd query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("nvd: %s: %s", resp.Status, string(body))
	}

	var result nvdResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("nvd decode: %w", err)
	}

	var cves []CVE
	for _, v := range result.Vulnerabilities {
		cve := v.CVE
		desc := ""
		for _, d := range cve.Descriptions {
			if d.Lang == "en" {
				desc = d.Value
				break
			}
		}
		sev, score := extractSeverity(cve.Metrics)
		cves = append(cves, CVE{
			ID:          cve.ID,
			Severity:    sev,
			Score:       score,
			Description: desc,
			URL:         "https://nvd.nist.gov/vuln/detail/" + cve.ID,
		})
	}

	c.cache.Set(cacheKey, cves)
	return cves, nil
}

func extractSeverity(m nvdMetrics) (string, float64) {
	if len(m.CVSSMetricV31) > 0 {
		return m.CVSSMetricV31[0].CVSSData.BaseSeverity, m.CVSSMetricV31[0].CVSSData.BaseScore
	}
	if len(m.CVSSMetricV30) > 0 {
		return m.CVSSMetricV30[0].CVSSData.BaseSeverity, m.CVSSMetricV30[0].CVSSData.BaseScore
	}
	if len(m.CVSSMetricV2) > 0 {
		score := m.CVSSMetricV2[0].CVSSData.BaseScore
		sev := "LOW"
		if score >= 7.0 {
			sev = "HIGH"
		} else if score >= 4.0 {
			sev = "MEDIUM"
		}
		return sev, score
	}
	return "UNKNOWN", 0
}

// QueryService runs NVD queries for a product+version pair.
func (c *NVDClient) QueryService(product, version string) ([]CVE, error) {
	if product == "" {
		return nil, nil
	}
	keyword := product
	if version != "" {
		keyword = product + " " + version
	}
	return c.Query(keyword)
}

// BatchQueryServices queries NVD for multiple services, deduplicating by CVE ID.
func (c *NVDClient) BatchQueryServices(services []ServiceFingerprint) ([]CVE, error) {
	seen := make(map[string]bool)
	var all []CVE

	for _, svc := range services {
		if svc.Product == "" {
			continue
		}
		cves, err := c.QueryService(svc.Product, svc.Version)
		if err != nil {
			// NVD timeouts are non-fatal; log and continue
			continue
		}
		for _, cve := range cves {
			if !seen[cve.ID] {
				seen[cve.ID] = true
				all = append(all, cve)
			}
		}
		// Rate limit: sleep 6s between requests to stay under 5 req/30s
		time.Sleep(6 * time.Second)
	}
	return all, nil
}

// ServiceFingerprint holds a product/version pair from nmap.
type ServiceFingerprint struct {
	Product string
	Version string
}

// FormatCVEsForPrompt returns a markdown string of CVEs for the AI prompt.
func FormatCVEsForPrompt(cves []CVE) string {
	if len(cves) == 0 {
		return "No known CVEs found in NVD for the discovered services."
	}
	var b strings.Builder
	b.WriteString("### Known CVEs (NVD Database)\n")
	for _, c := range cves {
		b.WriteString(fmt.Sprintf("- **%s** | Severity: %s (%.1f) | %s\n  %s\n", c.ID, c.Severity, c.Score, c.Description, c.URL))
	}
	return b.String()
}
