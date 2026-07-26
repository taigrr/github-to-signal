# Agent Guide for github-to-signal

HTTP server that receives GitHub webhook events and forwards them as Signal messages via signal-cli.

## Commands

```bash
# Build
GOWORK=off go build -o github-to-signal .

# Test
GOWORK=off go test .

# Run (requires config.toml)
./github-to-signal
```

**Note**: This repo may not be in a parent `go.work` workspace. Use `GOWORK=off` to ensure commands work standalone.

## Project Structure

```
.
├── main.go          # Entry point, HTTP server, event handlers
├── config.go        # Configuration loading (TOML + env vars)
├── filter.go        # Event filtering logic
├── filter_test.go   # Tests for event filtering
├── format.go        # Message formatting for each event type
├── config.example.toml
└── deploy/          # Systemd services and nginx config
```

All source files are in the root directory (single `main` package, no subdirectories).

## Configuration

Configuration via `config.toml` or environment variables with `GH2SIG_` prefix:

| Config Key | Env Variable | Description |
|------------|--------------|-------------|
| `webhook_secret` | `GH2SIG_WEBHOOK_SECRET` | GitHub webhook secret |
| `listen_addr` | `GH2SIG_LISTEN_ADDR` | Server address (default `:9900`) |
| `signal_url` | `GH2SIG_SIGNAL_URL` | signal-cli JSON-RPC endpoint |
| `signal_account` | `GH2SIG_SIGNAL_ACCOUNT` | Phone number for signal-cli |
| `signal_recipient` | `GH2SIG_SIGNAL_RECIPIENT` | Recipient UUID for DMs |
| `signal_group_id` | `GH2SIG_SIGNAL_GROUP_ID` | Group ID (overrides recipient) |
| `signal_cli_path` | `GH2SIG_SIGNAL_CLI_PATH` | Path to signal-cli binary; if set, the daemon is launched and supervised in-process |
| `signal_memory_limit_mb` | `GH2SIG_SIGNAL_MEMORY_LIMIT_MB` | RSS threshold (MiB) for the JVM memory watchdog; restarts the managed daemon when exceeded |
| `signal_java_max_heap_mb` | `GH2SIG_SIGNAL_JAVA_MAX_HEAP_MB` | Explicit JVM `-Xmx` cap (MiB); auto-derived from the limit if 0 |
| `events` | `GH2SIG_EVENTS` | Comma-separated event filter |

Configuration is loaded using [jety](https://github.com/taigrr/jety) library.

## Code Patterns

### Event Handlers

Each GitHub event type has:
1. A handler method on `notifier` struct in `main.go`
2. A `format*` function in `format.go` that returns the Signal message

Handler pattern:
```go
func (n *notifier) onEventName(ctx context.Context, _ string, _ string, event *github.EventType) error {
    if !n.filter.Allowed("event_name", event.GetAction()) {
        return nil
    }
    n.send(ctx, formatEventName(event))
    return nil
}
```

### Event Filtering

`EventFilter` in `filter.go` supports:
- Empty filter = allow all events
- `"event"` = all actions of that event type
- `"event:action"` = specific action only

Two-level check:
1. `EventEnabled(event)` — used at registration time to skip registering handlers
2. `Allowed(event, action)` — checked at runtime for action-level filtering

### Message Formatting

All `format*` functions in `format.go`:
- Return a string (empty string = no message sent)
- Use `[repo] user action ...` prefix format
- Include relevant URLs
- Truncate bodies with `truncate()` helper

### Dependencies

- `cbrgm/githubevents` — GitHub webhook event parsing and routing
- `google/go-github` — GitHub API types
- `taigrr/signalcli` — signal-cli JSON-RPC client
- `taigrr/jety` — Configuration (TOML/JSON/YAML/env)

## Adding New Event Types

1. Add handler method in `main.go` following existing pattern
2. Register handler in `main()` with `EventEnabled` check
3. Add `format*` function in `format.go`
4. Add event name to filter docs in `config.example.toml`

## Testing

Only `filter_test.go` exists — table-driven tests for `EventFilter`.

```bash
GOWORK=off go test -v .
```

## HTTP Endpoints

| Path | Method | Description |
|------|--------|-------------|
| `/webhook` | POST | GitHub webhook receiver |
| `/health` | GET | Health check (returns `ok`) |

## Deployment

Systemd services in `deploy/`:
- `signal-cli-bot.service` — runs signal-cli daemon
- `github-to-signal.service` — runs this server (depends on signal-cli)
- `github-to-signal.nginx.conf` — nginx reverse proxy config

The server expects signal-cli to be running on `127.0.0.1:8081`.

## Gotchas

- **GOWORK**: May need `GOWORK=off` if a parent `go.work` exists
- **signal-cli port**: Default in code is `8080`, but deployment uses `8081` to avoid conflicts
- **Workflow runs**: Only notifies on `completed` action, ignores `requested`/`in_progress`
- **Empty message**: Returning `""` from a formatter skips sending (used by workflow_run filter)
- **Managed daemon / memory leak**: signal-cli runs on the JVM and gradually leaks memory. When `signal_cli_path` is set, the server launches signal-cli via `signalcli.NewDaemon`, caps its heap (`JavaMaxHeapMB`), and runs `Daemon.Watch` to restart it once RSS exceeds `signal_memory_limit_mb`. Without `signal_cli_path`, the daemon is external (systemd) and the watchdog does not apply — bound memory via the unit's `MemoryMax` instead. See `daemon.go`.
