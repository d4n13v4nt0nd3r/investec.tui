# AGENTS.md

Guidelines for AI agents working on this codebase.

## Project Overview

Go TUI application for Investec Open Banking (Private Banking). Uses Bubble Tea for the terminal UI and calls the Investec REST API for account data.

## Tech Stack

- **Language:** Go
- **TUI framework:** charmbracelet/bubbletea
- **Styling:** charmbracelet/lipgloss
- **Env loading:** joho/godotenv

## Architecture

- `main.go` — entrypoint, loads `.env`, authenticates, starts Bubble Tea
- `internal/api/` — HTTP client and response models. All Investec API interaction lives here.
- `internal/tui/` — Bubble Tea views and styling. Each view is a separate file.

The app uses a single root `Model` in `app.go` that routes between three views: accounts → balance → transactions.

## Key Conventions

- **No mock data.** Always use real API responses. Do not add mock/stub data unless writing a temporary test.
- **Currency:** Never assume `$` or any currency symbol. Use the `currency` field returned by the API.
- **Number formatting:** Use space ` ` as thousand separator, show ≥2 decimal places. See `FormatAmount()` in `styles.go`.
- **Secrets:** The `.env` file contains credentials and is gitignored. Never log, print, or commit secrets.
- **Commits:** Keep commit messages short. Use `Co-Authored-By: Oz <oz-agent@warp.dev>` when AI-assisted.
- **No auto-push:** Do not run `git push` or merge unless explicitly asked.

## API Reference

Base URL: `https://openapi.investec.com`

### Auth
- `POST /identity/v2/oauth2/token` — OAuth2 client_credentials grant
- Basic Auth header: base64(client_id:client_secret)
- Header: `x-api-key`
- Token expires in ~30 min; client auto-refreshes

### Endpoints
- `GET /za/pb/v1/accounts` — list accounts
- `GET /za/pb/v1/accounts/{accountId}/balance` — account balance
- `GET /za/pb/v1/accounts/{accountId}/transactions?fromDate=&toDate=` — transactions (ISO 8601 dates)

## Build & Run

```bash
./run          # builds and runs
go build .     # build only
```

## Adding New Views

1. Create a new view struct and render method in `internal/tui/`
2. Add a `viewState` constant in `app.go`
3. Add message types for async data loading
4. Wire keyboard handling in `handleKey()`
5. Add rendering in `View()`

## Testing

No test framework is configured yet. When adding tests, use the standard `go test` tooling.
