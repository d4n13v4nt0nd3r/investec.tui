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

## Step 3: Create Your Credentials File

You need to create a small text file called `.env` that contains the list of countries you bank in, plus the API keys for each of them (from Step 1).

Credentials are per country, prefixed with the country code:

```
COUNTRY_LIST={South Africa:ZA;Mauritius:MU}

ZA_CLIENT_ID=paste_your_za_client_id_here
ZA_CLIENT_SECRET=paste_your_za_client_secret_here
ZA_API_KEY=paste_your_za_api_key_here

MU_CLIENT_ID=paste_your_mu_client_id_here
MU_CLIENT_SECRET=paste_your_mu_client_secret_here
MU_API_KEY=paste_your_mu_api_key_here
```

- `COUNTRY_LIST` is a `{Name:CODE;Name:CODE}` list. Only the countries listed here appear on the landing page.
- If you only bank in one country, list just that one.
- Legacy `INVESTEC_CLIENT_ID` / `INVESTEC_CLIENT_SECRET` / `INVESTEC_API_KEY` are still honoured as a fallback for any country that has no `<CODE>_*` values of its own.

### macOS

1. Open **TextEdit**
2. Go to **Format -> Make Plain Text** (this is important -- it must be plain text, not rich text)
3. Type the lines above, replacing the placeholder values with your actual keys from Step 1
4. Save the file as `.env` (the dot at the start is important) -- save it somewhere you can find it, like your Desktop
5. If macOS warns you about the dot in the filename, click **Use "."**

### Windows

1. Open **Notepad**
2. Type the lines above, replacing the placeholder values with your actual keys from Step 1
3. Go to **File -> Save As**
4. In the "Save as type" dropdown, select **All Files (*.*)**
5. Name the file `.env` (with the dot) and save it somewhere you can find it, like your Desktop

> **Note:** Make sure there are no spaces around the `=` signs.

## Step 4: Download and Run the App

Open your terminal (**Terminal** on Mac, **Git Bash** on Windows) and run these commands one at a time:

### macOS

```bash
# Download the app
git clone https://github.com/BeamMoney/tui.investec-openbanking.go.git

# Go into the app folder
cd tui.investec-openbanking.go

# Copy your .env file into the app folder (adjust the path to where you saved it)
cp ~/Desktop/.env .

# Start the app
./run
```

### Windows (Git Bash)

```bash
# Download the app
git clone https://github.com/BeamMoney/tui.investec-openbanking.go.git

# Go into the app folder
cd tui.investec-openbanking.go

# Copy your .env file into the app folder (adjust the path to where you saved it)
cp ~/Desktop/.env .

# Build and start the app
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
| `could not load .env file` | Make sure the `.env` file is in the same folder as the app |
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
├── .env                     # Credentials (gitignored)
├── run                      # Build & run script
└── docs/                    # Documentation
```

## Formatting

- Amounts display ≥2 decimal places
- Thousand separator is a space (e.g. `ZAR 12 345.67`)
- Currency is never assumed — always taken from the API response
