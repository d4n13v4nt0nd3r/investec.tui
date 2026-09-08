# AGENTS.md

Guidelines for AI agents working on this codebase.

## Project Overview

Go TUI application for Investec Open Banking (Private Banking). Uses Bubble Tea for the terminal UI and calls the Investec REST API for account data.

## Tech Stack

- **Language:** Go
- **TUI framework:** charmbracelet/bubbletea
- **Components:** charmbracelet/bubbles (`textinput`, for the masked credential fields)
- **Styling:** charmbracelet/lipgloss
- **Env loading:** joho/godotenv

## Architecture

- `main.go` — entrypoint, resolves and loads the credentials file, starts Bubble Tea
- `internal/config/` — `COUNTRY_LIST` parsing, per-country credential lookup, credentials-file discovery (`env_file.go`), and writing the file back out (`save.go`)
- `internal/api/` — HTTP client and response models. All Investec API interaction lives here.
- `internal/tui/` — Bubble Tea views and styling. Each view is a separate file.
- `internal/startup/` — platform glue for running as a downloaded app: Terminal relaunch on macOS, console pause on Windows, and the printed fallback for when there is no terminal to draw the setup screen on
- `scripts/release.sh` — builds, signs, notarizes and publishes the Mac/Windows packages
- `packaging/` — app icon artwork, the generated `.icns`/`.ico`, and the Python tools that build them. See `docs/releasing.md`; rebuild with `./icons`, which is only needed when the artwork changes.

The app uses a single root `Model` in `app.go` that routes between views: setup → country → accounts → balance → transactions / documents, plus a save-as overlay for PDF/CSV downloads. The API client is created (and authenticated) only after a country is selected.

`setup.go` is the guided credentials screen. It opens by itself when nothing is configured (`config.ErrNoCredentials`) and on `c` from the country page. Values are entered masked, checked against the API, then written by `config.SaveCountries`, which edits the existing file in place and never disturbs comments or settings it does not manage.

## Key Conventions

- **No mock data.** Always use real API responses. Do not add mock/stub data unless writing a temporary test.
- **Currency:** Never assume `$` or any currency symbol. Use the `currency` field returned by the API.
- **Number formatting:** Use space ` ` as thousand separator, show ≥2 decimal places. See `FormatAmount()` in `styles.go`.
- **Secrets:** The `.env` file contains credentials and is gitignored. Never log, print, or commit secrets, and never put a value in an error message. The file is written `0600` inside a `0700` folder, through a temp file and a rename.
- **Commits:** Keep commit messages short. Use `Co-Authored-By: Oz <oz-agent@warp.dev>` when AI-assisted.
- **No auto-push:** Do not run `git push` or merge unless explicitly asked.

## Configuration

The credentials file is resolved by `config.ResolveEnvFile()`, which tries `investec.env` then `.env` in: the `INVESTEC_TUI_ENV` path, the working directory, the executable's folder, and the per-user config folder (`~/Library/Application Support/investec-tui` on macOS, `%APPDATA%\investec-tui` on Windows). A missing override path deliberately does not fall back.

`config.TargetEnvFile()` is the write side: the file already in use, else the override path, else `investec.env` in the per-user folder.

`env.example` is the committed template; `.env` (gitignored) holds the country list and per-country credentials:

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
- `GET /{country}/pb/v1/accounts/{accountId}/documents?fromDate=&toDate=` — list PDF documents
- `GET /{country}/pb/v1/accounts/{accountId}/document/{documentType}/{documentDate}` — download PDF bytes

### Country differences
- ZA: `data.accounts[]`, balance at `data`, `data.transactions[]`, documents at `data[]`, string IDs, `type` = CREDIT/DEBIT, dates optional
- MU: `data.accounts.accounts[]`, balance at `data.accounts.balance`, `data.accounts.transactions[]`, documents at `availableDocuments.documentInformation[]`, numeric IDs, `creditAmount`/`debitAmount`, `fromDate`/`toDate` required (defaults to last 90 days)

Parsing for both shapes lives in `internal/api/models.go` (`parseAccounts`, `parseBalance`, `parseTransactions`, `parseDocuments`) and is covered by `internal/api/parse_test.go`. CSV/path helpers live in `internal/export`.

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
