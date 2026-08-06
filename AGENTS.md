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

- `main.go` — entrypoint, loads `.env` and the country list, starts Bubble Tea
- `internal/config/` — `COUNTRY_LIST` parsing and per-country credential lookup
- `internal/api/` — HTTP client and response models. All Investec API interaction lives here.
- `internal/tui/` — Bubble Tea views and styling. Each view is a separate file.

The app uses a single root `Model` in `app.go` that routes between four views: country → accounts → balance → transactions. The API client is created (and authenticated) only after a country is selected.

## Key Conventions

- **No mock data.** Always use real API responses. Do not add mock/stub data unless writing a temporary test.
- **Currency:** Never assume `$` or any currency symbol. Use the `currency` field returned by the API.
- **Number formatting:** Use space ` ` as thousand separator, show ≥2 decimal places. See `FormatAmount()` in `styles.go`.
- **Secrets:** The `.env` file contains credentials and is gitignored. Never log, print, or commit secrets.
- **Commits:** Keep commit messages short. Use `Co-Authored-By: Oz <oz-agent@warp.dev>` when AI-assisted.
- **No auto-push:** Do not run `git push` or merge unless explicitly asked.

## Configuration

`.env` holds the country list and per-country credentials:

```
COUNTRY_LIST={South Africa:ZA;Mauritius:MU}
ZA_CLIENT_ID= / ZA_CLIENT_SECRET= / ZA_API_KEY=
MU_CLIENT_ID= / MU_CLIENT_SECRET= / MU_API_KEY=
```

Legacy `INVESTEC_*` variables are a fallback for countries without `<CODE>_*` values.

## API Reference

Base URL: `https://openapi.investec.com`

### Auth
- `POST /identity/v2/oauth2/token` — OAuth2 client_credentials grant
- Basic Auth header: base64(client_id:client_secret)
- Header: `x-api-key`
- Token expires in ~30 min; client auto-refreshes

### Endpoints
The selected country code is the first path segment (`za`, `mu`, ...).
- `GET /{country}/pb/v1/accounts` — list accounts
- `GET /{country}/pb/v1/accounts/{accountId}/balance` — account balance
- `GET /{country}/pb/v1/accounts/{accountId}/transactions?fromDate=&toDate=` — transactions (ISO 8601 dates)

### Country differences
- ZA: `data.accounts[]`, balance at `data`, `data.transactions[]`, string IDs, `type` = CREDIT/DEBIT, dates optional
- MU: `data.accounts.accounts[]`, balance at `data.accounts.balance`, `data.accounts.transactions[]`, numeric IDs, `creditAmount`/`debitAmount`, `fromDate`/`toDate` required (defaults to last 90 days)

Parsing for both shapes lives in `internal/api/models.go` (`parseAccounts`, `parseBalance`, `parseTransactions`) and is covered by `internal/api/parse_test.go`.

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

Standard `go test ./...`. API response parsing is covered in `internal/api/parse_test.go`.
