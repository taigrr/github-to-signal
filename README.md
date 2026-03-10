# github-to-signal

HTTP server that receives GitHub webhook events and forwards them as Signal messages via [signal-cli](https://github.com/AsamK/signal-cli).

## Events

Push, issues, issue comments, pull requests, PR reviews, PR review comments, releases, stars, forks, workflow runs, branch/tag creation and deletion.

## Setup

1. Copy `config.example.toml` to `config.toml` and fill in values
2. Run the server: `go run .`
3. Add a webhook in your GitHub repo pointing to `https://your-host:9900/webhook`

All config values can also be set via environment variables with `GH2SIG_` prefix (e.g. `GH2SIG_SIGNAL_ACCOUNT`).

## Requirements

- [signal-cli](https://github.com/AsamK/signal-cli) running in daemon mode with JSON-RPC enabled
