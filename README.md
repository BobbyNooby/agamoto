# agamoto

A CLI orchestrator for nmap and OpenRouter. Run a scan, get CVE context from NVD and CISA KEV, then ask an LLM for attack recommendations with live web search. Made this to try out golang.

## What it does

1. Runs nmap with whatever flags you pass after `--`.
2. Parses the XML output and extracts discovered services (product + version).
3. Queries the **NVD** API for known CVEs.
4. Cross-references with **CISA KEV** for actively-exploited vulnerabilities.
5. Sends the nmap XML + research context to an OpenRouter-compatible LLM with the **web search plugin** enabled.
6. Streams back a readable attack plan and follow-up commands.

## Flow

```
[your args] ──► [agamoto]
                    │
                    ▼
            [run nmap scan]
                    │
                    ▼
            [parse XML output]
                    │
                    ▼
   [extract service fingerprints (product + version)]
                    │
           ┌────────┴────────┐
           ▼                 ▼
    [NVD CVE lookup]   [CISA KEV load]
    (keyword search    (cached JSON,
    by product+ver)    24h TTL)
           │                 │
           └────────┬────────┘
                    ▼
         [build research context]
                    │
                    ▼
         [AI prompt = nmap XML + research]
                    │
                    ▼
    [OpenRouter chat + web_search plugin]
                    │
                    ▼
         [stream AI analysis]
```

Research runs in two phases: NVD and CISA KEV are queried in parallel first, then the results are fed into the AI prompt. The web search plugin is enabled on the chat request itself, so the model can pull fresh references while generating the analysis.

## Prompts

Prompts live in [`internal/ai/prompts.go`](internal/ai/prompts.go). Edit them if you want the AI to focus on something else or change its style.

## Install

One-liner for macOS / Linux (installs nmap if missing, then installs agamoto):

```bash
curl -fsSL https://github.com/BobbyNooby/agamoto/raw/main/install.sh | sh
```

You need [Go](https://go.dev/dl/) installed. If you already have agamoto, the script will skip the install and tell you how to update.

To force a reinstall:

```bash
curl -fsSL https://github.com/BobbyNooby/agamoto/raw/main/install.sh | sh -s -- --force
```

Build from source:

```bash
git clone https://github.com/BobbyNooby/agamoto.git
cd agamoto
go build -o agamoto ./cmd/agamoto
```

Or run directly:

```bash
go run ./cmd/agamoto scan scanme.nmap.org -- -p 80 -sV
```

## Setup

Set an OpenRouter API key:

```bash
./agamoto config --api-key sk-or-v1-...
```

Optional:

```bash
./agamoto config --nvd-api-key <key>    # higher NVD rate limits
./agamoto config --model <model>        # default: deepseek/deepseek-v4-flash
./agamoto config --web-search-max-results 5
```

Config is stored in `~/.config/agamoto/config.json`.

Precedence: `defaults < config file < env vars < flags`

## Usage

### Scan

Everything after `--` is passed straight to nmap.

```bash
./agamoto scan scanme.nmap.org -- -p 80 -sV
./agamoto scan scanme.nmap.org -o report.txt -- -p- -sV -sC
./agamoto scan scanme.nmap.org -d -- -p 80 -sV
```

Short forms:

```bash
./agamoto s scanme.nmap.org -d -- -p 80 -sV
```

### Config

```bash
./agamoto config --api-key sk-or-v1-...
./agamoto config --model deepseek/deepseek-v4-flash
```

Short form:

```bash
./agamoto c --api-key sk-or-v1-...
```

## Example output

```bash
$ ./agamoto scan scanme.nmap.org -- -p 80 -sV

[agamoto] Target: scanme.nmap.org
[agamoto] nmap flags: [-p 80 -sV]
  → Running nmap -oX - --stats-every 5s -p 80 -sV scanme.nmap.org
[agamoto] nmap scan complete, parsing results...
[agamoto] Parsed 1 port(s) across 1 host(s)
[agamoto] Generating table report...
Host: 45.33.32.156 (up)
PORT     STATE  SERVICE              VERSION
------------------------------------------------------------
80/tcp   open   Apache httpd 2.4.7   2.4.7

[agamoto] Connecting to https://openrouter.ai/api/v1 (model: deepseek/deepseek-v4-flash)...
[agamoto] API key valid
[agamoto] Web search enabled (max 5 results)
[agamoto] NVD API key: not set (default 5 requests/30s rate limit)
[agamoto]   Loading CISA KEV catalog from cisa.gov...
[agamoto]   Querying NVD: "Apache httpd 2.4.7"
[agamoto]   CISA KEV catalog loaded (1647 entries, version 2026.07.16)
[agamoto]   NVD returned 1 CVE(s) for "Apache httpd 2.4.7"
[agamoto] CVE intelligence: 1 CVE(s), 0 KEV match(es)
[agamoto] Awaiting AI response... \

=== AI Analysis ===
🔍 Pentest Analysis Report: scanme.nmap.org

Command run: agamoto scan scanme.nmap.org -- -p 80 -sV
Target: 45.33.32.156 / scanme.nmap.org

─────────────────────────────────

1. Service Inventory

| Port | Status | Service | Version          | OS (from banner) |
|------|--------|---------|------------------|------------------|
| 80   | open   | HTTP    | Apache httpd 2.4.7 (Ubuntu) | Ubuntu |

2. Known Vulnerabilities & Exploits

🔴 CVE-2021-44224 | CVSS 8.2 (HIGH)
  Apache HTTP Server 2.4.7 – 2.4.51
  SSRF via forward proxy. Requires ProxyRequests On.

3. Network-Level Attack Opportunities

- MITM / cleartext traffic on HTTP
- SSRF chaining if proxy mode is enabled
- HTTP method abuse (PUT/DELETE/TRACE)

4. Recommended nmap Follow-Up Actions

nmap -sV -p- --script http-enum,http-vuln-cve2021-44224 scanme.nmap.org

5. Recommended agamoto Follow-Up Command

─── code block ───
agamoto scan scanme.nmap.org -o full_scan.txt -- -sV -sC -p- --script http-enum,http-vuln-cve2021-44224,http-methods
─── code block ───

[agamoto] Done.
```

## Commands

```
agamoto [command]

├── config [flags] (alias: c)
│   Manage agamoto configuration
│   Flags:
│     --api-base string              OpenAI-compatible base URL
│     --api-key string               OpenAI-compatible API key
│     --model string                 Model name
│     --nvd-api-key string           NVD API key (optional; higher rate limits)
│     --web-search-max-results int   Web search results per request (1-10)
│
└── scan <target> [-- <nmap-args>] [flags] (alias: s)
    Scan a target with nmap
    Flags:
      -d, --debug           Debug mode: show raw nmap XML, full AI prompt, and response metadata
      -h, --help            help for scan
      -n, --no-web-search   Disable web search (NVD + CISA KEV still run)
      -o, --output string   Write results to file
```

## Package managers

Not on homebrew / apt / etc yet — build from source for now.

## Testing

```bash
go test ./...
```

Uses a saved nmap XML fixture — no real nmap execution required.

## Future ideas

- Offline CVE cache for air-gapped use
- CVE → Exploit-DB / Metasploit cross-reference
- Export to markdown/pdf report
