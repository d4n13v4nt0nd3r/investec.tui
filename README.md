# Investec Open Banking TUI

[![Go Version](https://img.shields.io/github/go-mod/go-version/d4n13v4nt0nd3r/investec.tui)](https://go.dev/)
[![License](https://img.shields.io/github/license/d4n13v4nt0nd3r/investec.tui)](./LICENSE)
[![Release](https://img.shields.io/github/v/release/d4n13v4nt0nd3r/investec.tui)](https://github.com/d4n13v4nt0nd3r/investec.tui/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/d4n13v4nt0nd3r/investec.tui)](https://goreportcard.com/report/github.com/d4n13v4nt0nd3r/investec.tui)

A terminal UI for Investec Private Banking: accounts, balances, transactions, statements, and CSV export.

Built with Go, [Bubble Tea](https://github.com/charmbracelet/bubbletea), and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

![App icon sizes](packaging/source/preview.png)

## Features

- Multi-country landing page (South Africa, Mauritius, and any country you configure)
- Guided setup screen with masked credential entry and live API check
- Account list, balances, posted and pending transactions
- Fuzzy search on transaction description and amount
- Date filters, CSV export, and PDF statement / tax certificate download
- Amounts use the API currency, a space as thousands separator, and at least 2 decimal places

## Getting Started

### Pre-built binaries

Download the latest build from [Releases](https://github.com/d4n13v4nt0nd3r/investec.tui/releases/latest).

| Platform | File |
| -------- | ---- |
| macOS (Intel + Apple Silicon) | `InvestecTUI-<version>.dmg` |
| Windows x64 | `InvestecTUI-<version>-Windows-x64.exe` |
| Windows ARM64 | `InvestecTUI-<version>-Windows-ARM64.exe` |

Builds are currently unsigned. macOS needs Control-click -> **Open** the first time; Windows needs **More info** -> **Run anyway** once. Details are in the steps below.

### From source

Requires [Go](https://go.dev/dl/).

```bash
git clone https://github.com/d4n13v4nt0nd3r/investec.tui.git
cd investec.tui
./run
```

## Step 1: Get Your API Credentials

Each user needs their own Investec Open Banking API keys. Credentials are personal and must **never** be shared.

1. Open [Investec Online](https://login.secure.investec.com) and log in
2. Open **Programmable Banking**, then the **Open API** tab
3. If you see **Enroll**, click it once to enable API access
4. Copy the three values:
   - **Client ID**
   - **Client Secret**
   - **API Key (x-api-key)**

> **Important:** These credentials give access to your bank account data. Treat them like a password -- do not share them, email them, or paste them into chat.

## Step 2: Install the App

### macOS

1. Download `InvestecTUI-<version>.dmg` from [Releases](https://github.com/d4n13v4nt0nd3r/investec.tui/releases/latest)
2. Double-click the `.dmg` and drag **InvestecTUI** into **Applications**
3. Eject the disk image

### Windows

1. Download `InvestecTUI-<version>-Windows-x64.exe` (or the ARM64 build if needed)
2. Move it somewhere permanent, such as Desktop or Documents
3. Choose **Keep** if the browser warns that the file is uncommon

## Step 3: Run the App

### macOS

In **Applications**, Control-click **InvestecTUI** -> **Open** -> **Open** again. A Terminal window opens with the app.

Only the first launch needs that path. If double-click was blocked, use **System Settings** -> **Privacy & Security** -> **Open Anyway**.

### Windows

Double-click the `.exe`. On the blue **Windows protected your PC** screen, click **More info**, then **Run anyway**. Windows only asks once.

## Step 4: Enter Your Keys

The first run opens a setup screen:

1. Tick the countries you bank in
2. Paste Client ID, Client Secret, and API Key for each (hidden while typing; `ctrl+r` reveals)
3. The app checks the keys against Investec, then saves them

| System | Where they are saved |
|--------|----------------------|
| macOS | `~/Library/Application Support/investec-tui/investec.env` |
| Windows | `%APPDATA%\investec-tui\investec.env` |

Press `c` on the country page later to change or add keys.

> **Important:** treat that file like a password. Do not email it or store it in a shared folder.

### Writing the file yourself

```
COUNTRY_LIST={South Africa:ZA;Mauritius:MU}

ZA_CLIENT_ID=your_za_client_id
ZA_CLIENT_SECRET=your_za_client_secret
ZA_API_KEY=your_za_api_key

MU_CLIENT_ID=your_mu_client_id
MU_CLIENT_SECRET=your_mu_client_secret
MU_API_KEY=your_mu_api_key
```

- `COUNTRY_LIST` is `{Name:CODE;Name:CODE}`. Drop countries you do not use.
- No spaces around `=`.
- On Windows you can keep `investec.env` next to the `.exe`.

## Updating

### From a release build

Download the newest file from [Releases](https://github.com/d4n13v4nt0nd3r/investec.tui/releases/latest) and replace the previous install.

### From source

```bash
cd investec.tui
git pull
./run
```

`./run` rebuilds before launching. Your credentials file is gitignored and is not touched by `git pull`.

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Setup says a key failed | Press `e` and re-paste. Values live under Programmable Banking -> Open API. |
| Setup could not save | Path not writable, or set `INVESTEC_TUI_ENV` to a file you own. |
| `Authentication failed` | Press `c` on the country page and re-enter that country's keys. |
| Country shows `missing` | Press `c` (or Enter on that row) and fill in its keys. |
| No accounts | API access may not be enrolled yet -- revisit Step 1. |
| Windows protected your PC | **More info** -> **Run anyway** (once). |
| Windows window flashes and closes | Open PowerShell, drag the `.exe` in, press Enter to see the message. |
| macOS cannot verify the developer | Control-click -> **Open**, or **Open Anyway** in Privacy & Security. |
| macOS double-click does nothing | Install into **Applications** from the DMG, not from the mounted image. |

## Developer notes

Credentials resolution order (tries `investec.env`, then `.env` in each place):

1. `INVESTEC_TUI_ENV` if set
2. Current working directory
3. Folder holding the executable
4. Per-user config folder from Step 4

```bash
go test -race ./...
```

Publishing packages: [docs/releasing.md](docs/releasing.md).

## Navigation

### Country selection

| Key | Action |
|-----|--------|
| Up/Down or k/j | Navigate |
| Enter | Connect and load accounts |
| c | Change or add credentials |
| q | Quit |

### Setup

| Key | Action |
|-----|--------|
| Enter | Next |
| Shift+Tab | Previous |
| Space | Tick a country |
| ctrl+r | Show or hide the value |
| ctrl+s | Save without API check |
| Esc | Back a step |

### Accounts

| Key | Action |
|-----|--------|
| Up/Down or k/j | Navigate |
| Enter | View balance |
| r | Refresh |
| Esc | Back to countries |
| q | Quit |

### Balance

| Key | Action |
|-----|--------|
| t | Transactions |
| p | Pending transactions |
| d | Documents |
| r | Refresh |
| Esc | Back to accounts |

### Transactions

| Key | Action |
|-----|--------|
| Up/Down or k/j | Scroll |
| s | Search description / amount |
| f | Filter by date range |
| e | Export visible rows to CSV |
| r | Refresh |
| Esc | Clear search, or back to balance |

Search is case-insensitive and fuzzy on description (or bank reference) and amount (spaces in thousands ignored). Dates use `YYYY-MM-DD`.

### Documents

| Key | Action |
|-----|--------|
| Up/Down or k/j | Scroll |
| Enter | Download PDF |
| f | Filter by date |
| r | Refresh |
| Esc | Back to balance |

API keys need **View statements** / **View tax certificates** for PDF downloads.

### Save as

Default folder is Downloads; default name is the account number.

| Key | Action |
|-----|--------|
| Type | Edit filename |
| Tab | Filename vs folder list |
| Up/Down | Browse folders |
| Enter | Save or open folder |
| y / n | Confirm overwrite |
| Esc | Cancel |

## API endpoints

Base URL: `https://openapi.investec.com`

The selected country code is the first path segment (`za`, `mu`, ...).

| Endpoint | Description |
|----------|-------------|
| `POST /identity/v2/oauth2/token` | OAuth2 client_credentials |
| `GET /{country}/pb/v1/accounts` | List accounts |
| `GET /{country}/pb/v1/accounts/{id}/balance` | Balance |
| `GET /{country}/pb/v1/accounts/{id}/transactions` | Transactions |
| `GET /{country}/pb/v1/accounts/{id}/documents` | List documents |
| `GET /{country}/pb/v1/accounts/{id}/document/{type}/{date}` | Download PDF |

Response shapes differ by country and are normalised in `internal/api/models.go`. Mauritius requires `fromDate`/`toDate` on transactions; the app defaults to the last 90 days.

## Project structure

```
├── main.go
├── internal/
│   ├── api/       # HTTP client and per-country parsing
│   ├── config/    # Credentials discovery and save
│   ├── export/    # CSV helpers
│   ├── startup/   # Terminal / .app launch glue
│   └── tui/       # Bubble Tea views
├── scripts/release.sh
├── packaging/     # Icons and release artwork
├── env.example
├── run
└── docs/
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT -- see [LICENSE](LICENSE).
