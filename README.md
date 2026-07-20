# agamoto

> A minimal Go CLI wrapper around `nmap`. Run a scan, parse the XML, and print a readable table.

## Requirements

- [Go](https://go.dev/dl/)
- [nmap](https://nmap.org/download.html)

## Install

```bash
go build -o agamoto ./cmd/agamoto
```

## Usage

### Scan a target

Everything after `--` is passed directly to `nmap`:

```bash
# Scan default ports
./agamoto scan localhost

# Pass specific nmap flags
./agamoto scan localhost -- -p 22,80,443 -sV -v
./agamoto scan 10.0.0.1 -- -p 1-65535 -sV -A

# Save output to file
./agamoto scan scanme.nmap.org -o report.txt -- -p 22,80 -sV
```

### Configure defaults

Set API provider preferences (used once AI is wired in):

```bash
./agamoto config --api-key sk-or-v1-...
./agamoto config --api-base https://api.openai.com/v1
./agamoto config --model gpt-4o

# View current config
./agamoto config
```

Configuration is stored in `~/.config/agamoto/config.json`.

## Commands

```
agamoto scan <target> [flags] [-- <nmap-args>]

Flags:
  -o, --output FILE        Write results to file
      --no-deep-research   Skip fetching full articles
      --no-web-search      Skip web research

agamoto config             View configuration
agamoto config [flags]     Update configuration

Config flags:
      --api-key KEY        OpenAI-compatible API key (env: OPENAI_API_KEY)
      --api-base URL       OpenAI-compatible base URL (env: OPENAI_BASE_URL)
                           default: https://openrouter.ai/api/v1
      --model NAME         Model name (env: AI_MODEL)
                           default: deepseek/deepseek-v4-flash
```

## Configuration precedence

```
defaults < config file < environment variables < flags
```

## Common API bases

```bash
# OpenRouter (default)
--api-base https://openrouter.ai/api/v1

# OpenAI
--api-base https://api.openai.com/v1

# Groq
--api-base https://api.groq.com/openai/v1

# Ollama local
--api-base http://localhost:11434/v1
```

## Testing

```bash
go test ./...
```

Tests use a saved nmap XML fixture in `testdata/` — no real nmap execution required.
