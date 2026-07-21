# agamoto

> A single-binary network reconnaissance CLI. Wraps nmap, parses XML, prints readable tables.

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
./agamoto config
```

Stored in `~/.config/agamoto/config.json`.

## Commands

```
agamoto scan <target> [flags] [-- <nmap-args>]

Flags:
  -o, --output FILE        Write results to file
      --no-deep-research   (reserved)
      --no-web-search      (reserved)

agamoto config [flags]

Config flags:
      --api-key KEY        API key (env: OPENAI_API_KEY)
      --api-base URL       API base URL (env: OPENAI_BASE_URL)
                           default: https://openrouter.ai/api/v1
      --model NAME         Model name (env: AI_MODEL)
                           default: deepseek/deepseek-v4-flash
```

## Configuration precedence

```
defaults < config file < environment variables < flags
```

## Testing

```bash
go test ./...
```

Tests use a saved nmap XML fixture — no real nmap execution required.
