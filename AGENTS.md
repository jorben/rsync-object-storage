# AGENTS.md

This file provides guidelines for agentic coding agents working in this repository.

## Project Overview

Rsync Object Storage - A real-time file synchronization tool that watches local directories and syncs changes to remote S3-compatible object storage. Written in Go 1.21.

## Build Commands

```bash
# Build the binary
go build -o ros main.go

# Build with version info
go build -ldflags="-s -w" -o ros main.go

# Run locally
go run main.go -c ./config.yaml

# Download dependencies
go mod download

# Tidy dependencies
go mod tidy
```

## Testing

No dedicated test suite exists. When adding tests:

```bash
# Run all tests
go test ./...

# Run a single test file
go test -v ./helper/file_test.go

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Code Style Guidelines

### Imports

- Group imports: stdlib first, then third-party, then local packages
- Use blank line between groups
- Alias imports only when necessary (e.g., `conf "github.com/ldigit/config"`)

```go
import (
    "context"
    "fmt"
    "os"
    "path/filepath"

    "github.com/jorben/rsync-object-storage/config"
    "github.com/jorben/rsync-object-storage/log"
    "go.uber.org/zap"
)
```

### Naming Conventions

- **Packages**: lowercase, short, descriptive (e.g., `log`, `helper`, `kv`, `enum`)
- **Structs**: PascalCase (e.g., `Transfer`, `SyncConfig`, `OutputConfig`)
- **Functions**: PascalCase for exported, camelCase for unexported
- **Variables**: camelCase, use short names for loop variables
- **Constants**: PascalCase or camelCase depending on exported status
- **Interfaces**: Name based on method behavior (e.g., `Reader`, `Writer`)

### Error Handling

- Return errors as values, don't use exceptions
- Use `errors.New()` for sentinel errors and `fmt.Errorf()` with `%w` for wrapping
- Use `errors.Is()` to check error types
- Handle errors at appropriate level; log and continue or return up

```go
// Sentinel error in enum package
var ErrSkipTransfer = errors.New("skipped, it's not a error")

// Wrapping with context
if err := os.ReadFile(path); err != nil {
    return fmt.Errorf("failed to read config: %w", err)
}

// Checking errors
if errors.Is(err, enum.ErrSkipTransfer) {
    log.Debugf("Skipping %s", path)
}
```

### Logging

- Use the `log` package wrapper around uber-go/zap
- Prefer structured logging with `log.Infof`, `log.Debugf`, `log.Errorf`
- Log with context: include relevant fields (file paths, operation types)
- Use `log.Fatalf` only for startup/config errors

### Concurrency

- Use channels for communication between goroutines
- Use `context.Context` for cancellation and timeouts
- Always defer channel closure when appropriate
- Use `sync.Mutex` for protecting shared state (see `kv/kv.go`)

```go
type Transfer struct {
    LocalPrefix  string
    RemotePrefix string
    HotDelay     time.Duration
    PutChan      chan string
    DeleteChan   chan string
    Storage      *Storage
}

func (t *Transfer) Run(ctx context.Context) {
    for {
        select {
        case path := <-t.PutChan:
            // handle put
        case path := <-t.DeleteChan:
            // handle delete
        }
    }
}
```

### Structs and Types

- Use struct tags for YAML serialization (see `config/config.go`)
- Keep structs focused; prefer composition over deep nesting
- Use time.Duration for time-based values in structs (config uses int, converted at runtime)
- Use meaningful field names that describe purpose

### Constants and Enums

- Define related constants in `enum/` package
- Use typed string constants for strategy options (e.g., `SymlinkSkip`, `SymlinkAddr`, `SymlinkFile`)
- Define sentinel errors as package-level variables (e.g., `ErrSkipTransfer`)

### Context Usage

- Pass `context.Context` as first parameter to functions performing I/O
- Use `context.Background()` for top-level initialization
- Check `ctx.Done()` in long-running loops

### File Organization

- One public type per file unless types are tightly coupled
- Group related functionality in same package
- Keep `main.go` minimal; delegate to package functions
- Helper utilities in `helper/` package
- Core components in root: `storage.go` (S3 operations), `transfer.go` (sync worker), `watcher.go` (file monitoring), `checkjob.go` (periodic reconciliation)

### Configuration

- Use the `github.com/ldigit/config` library for YAML config loading
- Validate and normalize config values after loading (see `config/config.go:90-119`)
- Provide `GetString()` method for config debugging

### Code Comments

- Comment exported types and functions
- Use Chinese comments when original comments are Chinese (match existing style)
- Keep comments concise and factual

### Secrets Handling

- Never log full secrets; use `helper.HideSecret()` for display (see `helper/string.go`)
- Use environment variables for sensitive credentials
