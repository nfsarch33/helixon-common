# helixon-common

Shared Go packages for the Helixon agent platform ecosystem.

## Packages

| Package | Purpose |
|---------|---------|
| `ndjson` | Thread-safe NDJSON writer with event helpers |
| `health` | HTTP health/readiness probe handlers |
| `slogx` | Structured logging setup with env-based level |
| `engram` | Engram memory engine HTTP client |
| `agentrace` | Agent trace event emission (NDJSON) |

## Install

```bash
go get github.com/nfsarch33/helixon-common
```

## Usage

```go
import (
    "github.com/nfsarch33/helixon-common/ndjson"
    "github.com/nfsarch33/helixon-common/health"
    "github.com/nfsarch33/helixon-common/slogx"
    "github.com/nfsarch33/helixon-common/engram"
    "github.com/nfsarch33/helixon-common/agentrace"
)
```

## License

MIT
