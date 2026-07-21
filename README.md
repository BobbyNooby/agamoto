# agamoto

> A single-binary network reconnaissance CLI. Wraps nmap, enriches results with CVE and open-source intelligence, and generates AI-driven attack recommendations.

## Requirements

- [Go](https://go.dev/dl/)
- [nmap](https://nmap.org/download.html)

## Install

```bash
go build -o agamoto ./cmd/agamoto
```

## Usage

### Scan a target

Everything after `--` is passed directly to nmap:

```bash
./agamoto scan localhost
./agamoto scan localhost -- -p 22,80,443 -sV
./agamoto scan scanme.nmap.org -o report.txt -- -p 22,80 -sV
```

### Configure defaults

```bash
./agamoto config --api-key sk-or-v1-...
./agamoto config --nvd-api-key <key>      # optional; removes NVD rate limits
./agamoto config --max-research-passes 3  # default 3
./agamoto config --max-urls-per-query 5   # default 5
./agamoto config
```

Stored in `~/.config/agamoto/config.json`.

## Commands

```
agamoto scan <target> [flags] [-- <nmap-args>]

Flags:
  -o, --output FILE        Write results to file
      --no-deep-research   Skip fetching full articles; use DDG snippets only
      --no-web-search      Skip DuckDuckGo web search (NVD + CISA KEV still run)

agamoto config [flags]

Config flags:
      --api-key KEY              API key (env: OPENAI_API_KEY)
      --api-base URL             API base URL (env: OPENAI_BASE_URL)
                                   default: https://openrouter.ai/api/v1
      --model NAME               Model name (env: AI_MODEL)
                                   default: deepseek/deepseek-v4-flash
      --nvd-api-key KEY          NVD API key (env: NVD_API_KEY)
      --max-research-passes N    Maximum deep-research passes (env: AGAMOTO_MAX_RESEARCH_PASSES)
      --max-urls-per-query N     Maximum URLs to fetch per DDG query (env: AGAMOTO_MAX_URLS_PER_QUERY)
```

## Research pipeline

For every scan, agamoto gathers intelligence from multiple sources:

1. **NVD (National Vulnerability Database)** — known CVEs for discovered services
2. **CISA KEV (Known Exploited Vulnerabilities)** — which CVEs are actively exploited in the wild
3. **DuckDuckGo** — recent breaches, advisories, discussions, and exploit references

By default, deep research is enabled:

- Up to 3 iterative DDG passes
- Full article text extraction from the top 5 results per query
- Follow-up queries generated from CVEs and initial findings

Use `--no-deep-research` to fetch only search snippets, and `--no-web-search` to skip DuckDuckGo entirely.

## Configuration precedence

```
defaults < config file < environment variables < flags
```

## Future enhancements

- **SearXNG backend** — self-hosted, multi-engine search with no rate limits
- **Tavily API backend** — AI-native search with structured results
- **Tongyi DeepResearch** — agentic research loop with synthesized citations
- **Offline CVE cache** — periodically mirror NVD and CISA KEV for air-gapped use
- **CVE exploit-DB / Metasploit cross-reference** — link CVEs to public exploit code

## Testing

```bash
go test ./...
```

Tests use a saved nmap XML fixture — no real nmap execution required.
