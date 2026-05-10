# webhookhub

The official CLI for [WebhookHub](https://webhookhub.dev). Stream webhooks captured at your stable WebhookHub URL down to your local development server in real time.

## Status

`v0.1.0-alpha` — auth and endpoint listing work. Streaming `forward` ships in v0.2.0.

## Install

```bash
# Coming soon: brew, scoop, install.sh.
go install github.com/shire-studio/webhookhub-cli/cmd/webhookhub@latest
```

## Quickstart

```bash
webhookhub auth login              # Paste your token from webhookhub.dev/profile/access-tokens
webhookhub auth whoami             # Confirm the active token
webhookhub endpoints               # List your endpoints
```

## Configuration

- Auth token stored at `$XDG_CONFIG_HOME/webhookhub/config.json` (mode 0600).
- Override the API base URL via `WEBHOOKHUB_API_URL` (default `https://webhookhub.dev`).

## Development

```bash
go test ./...
golangci-lint run ./...
```

## License

MIT — see `LICENSE`.
