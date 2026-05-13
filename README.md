# webhookhub

The official CLI for [WebhookHub](https://webhookhub.dev). Stream webhooks captured at your stable WebhookHub URL down to your local development server in real time. No tunnel domains, no rotating URLs, no proxy in the middle of your request path.

## Status

`v0.2.0-alpha` — auth, endpoint listing, and webhook forwarding all work. Distribution (Homebrew, Scoop, install.sh) lands in the next release.

## Install

```bash
# Until distribution lands, install from source:
go install github.com/shire-studio/webhookhub-cli/cmd/webhookhub@latest
```

## Quickstart

```bash
webhookhub auth login                                    # paste your token from webhookhub.dev/profile/access-tokens
webhookhub endpoints                                     # see your endpoint slugs
webhookhub forward stripe-test --to http://localhost:3000
```

## Multi-endpoint mode

Drop a `webhookhub.yaml` in your project root:

```yaml
forwards:
  - endpoint: stripe-test
    to: http://localhost:3000
  - endpoint: github
    to: http://localhost:3001/webhooks/github
```

Then `webhookhub forward` (no args) tunnels every mapping concurrently over a single connection. Log lines are prefixed with the slug.

## Flags

```
--to <url>          Local URL to replay against (single-endpoint mode)
--timeout <dur>     Per-replay timeout, default 30s. Examples: --timeout=10s, --timeout=1m
--verbose           Log request/response headers and bodies (off by default — webhooks contain secrets)
```

## Configuration

- Auth token stored at `os.UserConfigDir()/webhookhub/config.json` (mode 0600).
  - Linux: `~/.config/webhookhub/config.json`
  - macOS: `~/Library/Application Support/webhookhub/config.json`
  - Windows: `%AppData%\webhookhub\config.json`
- Forwards config: `./webhookhub.yaml` (preferred) or `$UserConfigDir/webhookhub/forwards.yaml`.
- Override the API base URL with `WEBHOOKHUB_API_URL` (default `https://webhookhub.dev`).

## Connection lifecycle

- Reconnects on transient network errors with backoff: 1s → 2s → 5s → 10s → 30s.
- A revoked token (401) is terminal — exits non-zero with a clear message; re-auth with `webhookhub auth login`.
- Webhooks captured while the CLI is offline are NOT auto-replayed when it reconnects (avoids surprise duplicates). Re-forward manually from the dashboard.

## Development

```bash
go test -race ./...
golangci-lint run ./...
```

## License

MIT — see `LICENSE`.
