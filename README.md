# Investec Open Banking TUI

A terminal UI for viewing Investec Private Banking account information, balances, and transactions.

Built with Go, [Bubble Tea](https://github.com/charmbracelet/bubbletea), and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Step 1: Get Your API Credentials

Each user needs their own Investec Open Banking API keys. Credentials are personal and must **never** be shared with others.

1. Open your web browser and go to [Investec Online](https://login.secure.investec.com)
2. Log in with your normal Investec username and password
3. Once logged in, look for **Programmable Banking** in the menu and click on it
4. Click the **Open API** tab at the top
5. If you see an **Enroll** button, click it to enable API access (you only need to do this once)
6. You should now see three values on the screen:
   - **Client ID** — a long string of letters and numbers
   - **Client Secret** — a shorter secret string
   - **API Key (x-api-key)** — a long encoded string
7. Copy each of these values and keep them somewhere safe (you'll need them in Step 3)

> **Important:** These credentials give access to your bank account data. Treat them like a password — do not share them, email them, or paste them into chat.

## Step 2: Install Required Software

You need two things installed on your computer: **Git** (to download the app) and **Go** (to build and run it).

### macOS

1. Open the **Terminal** app (press `Cmd + Space`, type `Terminal`, press Enter)
2. Install Homebrew (a package manager) by pasting this command and pressing Enter:
   ```bash
   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
   ```
   Follow the on-screen prompts. You may need to enter your Mac password.
3. Once Homebrew is installed, install Git and Go:
   ```bash
   brew install git go
   ```

### Windows

1. **Install Git:** Download from [git-scm.com/download/win](https://git-scm.com/download/win) and run the installer. Accept all the default options.
2. **Install Go:** Download from [go.dev/dl](https://go.dev/dl/), choose the **Windows** `.msi` installer, and run it. Accept the defaults.
3. After both installations, close and reopen any terminal windows so the new programs are recognised.
4. Open **Git Bash** (search for it in the Start menu) — use this for all the commands below.

## Step 3: Download the App

Open your terminal (**Terminal** on Mac, **Git Bash** on Windows) and run these commands one at a time:

```bash
# Download the app
git clone https://github.com/BeamMoney/tui.investec-openbanking.go.git

# Go into the app folder
cd tui.investec-openbanking.go
```

Stay in this folder for the remaining steps.

## Step 4: Create Your Credentials File

The app reads a file called `.env` in the project root (the folder you are now in). Start from the template that ships with the app:

```bash
cp env.example .env
nano .env
```

`nano` is a terminal text editor. Fill in your keys from Step 1 so the file looks like this:

```
COUNTRY_LIST={South Africa:ZA;Mauritius:MU}

ZA_CLIENT_ID=your_za_client_id
ZA_CLIENT_SECRET=your_za_client_secret
ZA_API_KEY=your_za_api_key

MU_CLIENT_ID=your_mu_client_id
MU_CLIENT_SECRET=your_mu_client_secret
MU_API_KEY=your_mu_api_key
```

To save and exit nano: press `Ctrl + O`, then `Enter`, then `Ctrl + X`.

Notes:

- `COUNTRY_LIST` is a `{Name:CODE;Name:CODE}` list. Only the countries listed here appear on the landing page, so delete any country you do not bank in (and its keys).
- Each country needs its own credentials from Step 1, obtained while logged in to that country's Investec Online profile.
- Do not put spaces around the `=` signs.
- `.env` is gitignored, so your keys stay on your machine.

Check which values are filled in without printing your secrets:

```bash
awk -F= '/^[A-Z_]+=/{print $1": "(length($2)>0?"set":"EMPTY")}' .env
```

## Step 5: Run the App

### macOS

```bash
./run
```

### Windows (Git Bash)

```bash
go build -o investec.openbanking.tui.exe .
./investec.openbanking.tui.exe
```

## Running the App Again Later

Once installed, you only need to do this each time:

### macOS

```bash
cd tui.investec-openbanking.go
./run
```

### Windows (Git Bash)

```bash
cd tui.investec-openbanking.go
./investec.openbanking.tui.exe
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `command not found: go` | Go is not installed or the terminal needs to be reopened after installation |
| `command not found: git` | Git is not installed |
| `Authentication failed` | Double-check the values in your `.env` file match exactly what Investec shows |
| `could not load .env file` | Make sure `.env` is in the project root, next to `main.go` and `run` |
| `no credentials found in .env` | The `<CODE>_*` values are still empty -- run `nano .env` again |
| A country shows `missing` on the landing page | That country's `<CODE>_CLIENT_ID`, `<CODE>_CLIENT_SECRET` or `<CODE>_API_KEY` is empty |
| App shows no accounts | Your API access may not be enrolled yet — revisit Step 1 |

## Navigation

### Country Selection (landing page)

| Key       | Action              |
|-----------|---------------------|
| ↑/↓ or k/j | Navigate countries |
| Enter     | Connect and load accounts |
| q         | Quit                |

### Accounts List

| Key       | Action              |
|-----------|---------------------|
| ↑/↓ or k/j | Navigate accounts |
| Enter     | View balance        |
| r         | Refresh             |
| Esc       | Back to country selection |
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

The country code selected on the landing page becomes the first path segment (`za`, `mu`, ...).

| Endpoint | Description |
|----------|-------------|
| `POST /identity/v2/oauth2/token` | OAuth2 client_credentials auth |
| `GET /{country}/pb/v1/accounts` | List accounts |
| `GET /{country}/pb/v1/accounts/{id}/balance` | Account balance |
| `GET /{country}/pb/v1/accounts/{id}/transactions` | Transaction history |

Base URL: `https://openapi.investec.com`

Response shapes differ per country and are normalised in `internal/api/models.go`:

- ZA: `data.accounts[]`, `data` (balance), `data.transactions[]`, string IDs, `type` of CREDIT/DEBIT
- MU: `data.accounts.accounts[]`, `data.accounts.balance`, `data.accounts.transactions[]`, numeric IDs, separate `creditAmount`/`debitAmount`

MU requires an explicit `fromDate`/`toDate` on transactions, so the app defaults to the last 90 days.

## Project Structure

```
├── main.go                  # Entrypoint
├── internal/
│   ├── api/
│   │   ├── client.go        # HTTP client, auth, country-scoped API methods
│   │   └── models.go        # Response structs and per-country parsing
│   ├── config/
│   │   └── config.go        # COUNTRY_LIST and per-country credentials
│   └── tui/
│       ├── app.go           # Bubble Tea model, routing
│       ├── country.go       # Country selection landing page
│       ├── accounts.go      # Accounts list view
│       ├── balance.go       # Balance detail view
│       ├── transactions.go  # Transactions table view
│       └── styles.go        # Styling and formatting
├── env.example              # Template for .env
├── .env                     # Credentials (gitignored)
├── run                      # Build & run script
└── docs/                    # Documentation
```

## Formatting

- Amounts display ≥2 decimal places
- Thousand separator is a space (e.g. `ZAR 12 345.67`)
- Currency is never assumed — always taken from the API response
