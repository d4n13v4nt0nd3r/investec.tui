# Investec Open Banking TUI

A terminal UI for viewing Investec Private Banking account information, balances, and transactions.

Built with Go, [Bubble Tea](https://github.com/charmbracelet/bubbletea), and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Setup

### Prerequisites

- Go 1.21+
- Investec Open Banking API credentials ([enrol here](https://login.secure.investec.com))

### Configuration

Create a `.env` file in the project root:

```
INVESTEC_CLIENT_ID=your_client_id
INVESTEC_CLIENT_SECRET=your_client_secret
INVESTEC_API_KEY=your_api_key
```

> **Note:** `.env` is gitignored and must never be committed.

### Run

```bash
./run
```

Or manually:

```bash
go build -o investec.openbanking.tui .
./investec.openbanking.tui
```

## Navigation

### Accounts List (default)

| Key       | Action              |
|-----------|---------------------|
| ↑/↓ or k/j | Navigate accounts |
| Enter     | View balance        |
| r         | Refresh             |
| q         | Quit                |

### Balance View

| Key   | Action             |
|-------|--------------------|
| t     | View transactions  |
| r     | Refresh balance    |
| Esc   | Back to accounts   |

### Transactions View

| Key       | Action                      |
|-----------|-----------------------------|
| ↑/↓ or k/j | Scroll transactions       |
| f         | Filter by date range        |
| r         | Refresh with current filter |
| Esc       | Back to balance             |

When filtering dates, type in `YYYY-MM-DD` format. Press Enter to confirm each field (from → to), then transactions reload automatically.

## API Endpoints Used

| Endpoint | Description |
|----------|-------------|
| `POST /identity/v2/oauth2/token` | OAuth2 client_credentials auth |
| `GET /za/pb/v1/accounts` | List accounts |
| `GET /za/pb/v1/accounts/{id}/balance` | Account balance |
| `GET /za/pb/v1/accounts/{id}/transactions` | Transaction history |

Base URL: `https://openapi.investec.com`

## Project Structure

```
├── main.go                  # Entrypoint
├── internal/
│   ├── api/
│   │   ├── client.go        # HTTP client, auth, API methods
│   │   └── models.go        # Response structs
│   └── tui/
│       ├── app.go           # Bubble Tea model, routing
│       ├── accounts.go      # Accounts list view
│       ├── balance.go       # Balance detail view
│       ├── transactions.go  # Transactions table view
│       └── styles.go        # Styling and formatting
├── .env                     # Credentials (gitignored)
├── run                      # Build & run script
└── docs/                    # Documentation
```

## Formatting

- Amounts display ≥2 decimal places
- Thousand separator is a space (e.g. `ZAR 12 345.67`)
- Currency is never assumed — always taken from the API response
