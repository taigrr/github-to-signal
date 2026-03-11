# github-to-signal

HTTP server that receives GitHub webhook events and forwards them as Signal messages via [signal-cli](https://github.com/AsamK/signal-cli).

## Supported Events

- **Push** — commits pushed to a branch
- **Issues** — opened, closed, reopened, etc.
- **Issue comments** — new comments on issues
- **Pull requests** — opened, closed, merged, etc.
- **PR reviews** — approved, changes requested, commented
- **PR review comments** — inline code comments on PRs
- **Releases** — published, drafted, etc.
- **Stars** — starred/unstarred (with count)
- **Forks** — repo forked
- **Workflow runs** — CI completed (with pass/fail indicator)
- **Branch/tag create** — new branches or tags
- **Branch/tag delete** — deleted branches or tags

## Setup

### 1. Signal Profile

Register a phone number with signal-cli, then set up the bot profile:

```bash
# Set the bot's display name
signal-cli -a +1YOURNUMBER updateProfile --given-name "Github" --family-name "PRs"

# Set the octocat avatar (included in assets/)
signal-cli -a +1YOURNUMBER updateProfile --avatar assets/octocat.png
```

### 2. Run signal-cli daemon

```bash
signal-cli -a +1YOURNUMBER daemon --http 127.0.0.1:8081 --no-receive-stdout
```

### 3. Configure

Copy `config.example.toml` to `config.toml`:

```toml
# GitHub webhook secret (set in your GitHub webhook settings)
webhook_secret = "your-secret-here"

# Address to listen on
listen_addr = ":9900"

# signal-cli JSON-RPC endpoint
signal_url = "http://127.0.0.1:8081"

# signal-cli account (phone number registered with signal-cli)
signal_account = "+1YOURNUMBER"

# Signal recipient UUID to send notifications to
signal_recipient = "your-uuid-here"
```

All values can also be set via environment variables with `GH2SIG_` prefix:

```bash
export GH2SIG_WEBHOOK_SECRET="your-secret"
export GH2SIG_SIGNAL_ACCOUNT="+1YOURNUMBER"
export GH2SIG_SIGNAL_RECIPIENT="recipient-uuid"
```

### 4. Build and run

```bash
go build -o github-to-signal .
./github-to-signal
```

### 5. Add GitHub webhook

In your repo (or org) settings:

1. Go to **Settings > Webhooks > Add webhook**
2. **Payload URL:** `https://your-host:9900/webhook`
3. **Content type:** `application/json`
4. **Secret:** same value as `webhook_secret` in your config
5. **Events:** select the events you want, or "Send me everything"

### Endpoints

| Path | Method | Description |
|------|--------|-------------|
| `/webhook` | POST | GitHub webhook receiver |
| `/health` | GET | Health check (returns `ok`) |

## Deployment

Systemd services and nginx config are in `deploy/`.

```bash
# Create service user
sudo useradd -r -m -s /usr/sbin/nologin signal-bot

# Install binary
go build -o /usr/local/bin/github-to-signal .

# Install config
sudo mkdir -p /etc/github-to-signal
sudo cp config.toml /etc/github-to-signal/
sudo chown -R signal-bot:signal-bot /etc/github-to-signal

# Install systemd services
sudo cp deploy/signal-cli-bot.service /etc/systemd/system/
sudo cp deploy/github-to-signal.service /etc/systemd/system/
sudo systemctl daemon-reload

# Enable and start (signal-cli-bot starts automatically as a dependency)
sudo systemctl enable --now github-to-signal

# Install nginx config
sudo cp deploy/github-to-signal.nginx.conf /etc/nginx/sites-available/github-to-signal
sudo ln -s /etc/nginx/sites-available/github-to-signal /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

Edit the service files first to set your phone number and paths. The signal-cli daemon listens on `127.0.0.1:8081` (not 8080, to avoid conflicts). Update `signal_url` in your config.toml to match.

## Dependencies

- [cbrgm/githubevents](https://github.com/cbrgm/githubevents) — GitHub webhook event handling
- [taigrr/signalcli](https://github.com/taigrr/signalcli) — signal-cli Go client
- [taigrr/jety](https://github.com/taigrr/jety) — configuration (TOML/JSON/YAML/env)

## License

0BSD
