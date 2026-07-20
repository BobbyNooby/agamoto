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

```bash
./agamoto scan <target>
./agamoto scan scanme.nmap.org -p 22,80,443
./agamoto scan localhost -p 1-65535 -v
```

## Flags

```
-p, --ports   Port range (default: 21-23,25,53,80,443,8080)
-v, --verbose Include closed/refused ports
-o, --output  Write results to file
```

## Testing

```bash
go test ./...
```

Tests use a saved nmap XML fixture in `testdata/` — no real nmap execution required.
