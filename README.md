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

## Step 2: Download the App

Nothing needs to be installed first — the download is a single self-contained file.

Go to the [Releases page](https://github.com/BeamMoney/tui.investec-openbanking.go/releases) (sign in to GitHub with your BeamMoney account) and grab the newest:

### macOS

1. Download `InvestecTUI-<version>.dmg`. One file works on both Intel and Apple Silicon Macs.
2. Double-click the downloaded `.dmg`.
3. Drag **InvestecTUI** onto the **Applications** folder shown in the window.
4. Eject the disk image (click the eject arrow next to it in Finder's sidebar).

### Windows

1. Download `InvestecTUI-<version>-Windows-x64.exe`. Choose the `ARM64` file instead only if you have a Windows-on-ARM machine such as a Surface Pro X.
2. Move it somewhere permanent, such as your **Desktop** or **Documents** folder.

   Windows will warn you that the file is not commonly downloaded, because we do not yet pay for a Microsoft signing certificate. Choose **Keep** to complete the download.

## Step 3: Create Your Credentials File

Create a plain text file containing your keys from Step 1:

```
COUNTRY_LIST={South Africa:ZA;Mauritius:MU}

ZA_CLIENT_ID=your_za_client_id
ZA_CLIENT_SECRET=your_za_client_secret
ZA_API_KEY=your_za_api_key

MU_CLIENT_ID=your_mu_client_id
MU_CLIENT_SECRET=your_mu_client_secret
MU_API_KEY=your_mu_api_key
```

Notes:

- `COUNTRY_LIST` is a `{Name:CODE;Name:CODE}` list. Only the countries listed here appear on the landing page, so delete any country you do not bank in (and its keys).
- Each country needs its own credentials from Step 1, obtained while logged in to that country's Investec Online profile.
- Do not put spaces around the `=` signs.

Save it as **`investec.env`** in the folder for your system:

| System | Where to save `investec.env` |
|--------|------------------------------|
| macOS | `~/Library/Application Support/investec-tui/` |
| Windows | `%APPDATA%\investec-tui\` |

Don't worry about finding that folder by hand. Run the app once (Step 4) and it will create the folder, open it for you, and print the exact path to save the file to.

On Windows you can also simply keep `investec.env` in the same folder as the `.exe`.

> **Important:** treat this file like a password. Do not email it around or store it in a shared folder.

## Step 4: Run the App

### macOS

Open your **Applications** folder and double-click **InvestecTUI**. A Terminal window opens and the app runs inside it.

### Windows

Double-click the `.exe` you saved. A console window opens and the app runs inside it.

The first time you run it, Windows shows a blue **"Windows protected your PC"** screen. This is expected for an unsigned app. Click **More info**, then **Run anyway**. Windows only asks once.

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Setup screen says no credentials file was found | Save `investec.env` in the folder the app printed, then run it again. The list of places it looked is shown on that screen. |
| `no credentials found in .env` | The `<CODE>_*` values in your file are still empty |
| `Authentication failed` | Double-check the values in your credentials file match exactly what Investec shows |
| A country shows `missing` on the landing page | That country's `<CODE>_CLIENT_ID`, `<CODE>_CLIENT_SECRET` or `<CODE>_API_KEY` is empty |
| App shows no accounts | Your API access may not be enrolled yet — revisit Step 1 |
| Windows: "Windows protected your PC" | Click **More info** -> **Run anyway**. Only needed once. |
| Windows: the window flashes and disappears | It should now pause and wait for Enter. If it does not, open PowerShell, drag the `.exe` into the window and press Enter to see the message. |
| macOS: nothing happens on double-click | Make sure you dragged the app to **Applications** from the disk image rather than running it from the mounted image |

## For Developers: Build From Source

Requires [Go](https://go.dev/dl/) and Git.

```bash
git clone https://github.com/BeamMoney/tui.investec-openbanking.go.git
cd tui.investec-openbanking.go
cp env.example .env
$EDITOR .env
./run
```

The app looks for a credentials file in this order, stopping at the first hit. In each folder it tries `investec.env` first, then `.env`:

1. The full path in the `INVESTEC_TUI_ENV` environment variable, if set
2. The current working directory
3. The folder holding the executable
4. The per-user config folder from the table in Step 3

So `./run` from a clone keeps using the repo's own `.env`, and a packaged build falls through to the per-user folder.

Running tests:

```bash
go test -race ./...
```

Publishing a release is documented in [docs/releasing.md](docs/releasing.md).

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
│   │   ├── config.go        # COUNTRY_LIST and per-country credentials
│   │   └── env_file.go      # Where the credentials file is looked for
│   ├── startup/
│   │   ├── console.go       # Terminal detection, press-Enter pause
│   │   ├── relaunch_*.go    # macOS: hand a double-clicked .app to Terminal
│   │   └── setup.go         # First-run "save your credentials here" screen
│   └── tui/
│       ├── app.go           # Bubble Tea model, routing
│       ├── country.go       # Country selection landing page
│       ├── accounts.go      # Accounts list view
│       ├── balance.go       # Balance detail view
│       ├── transactions.go  # Transactions table view
│       └── styles.go        # Styling and formatting
├── scripts/
│   └── release.sh           # Build, sign, notarize and publish the packages
├── env.example              # Template for the credentials file
├── .env                     # Credentials (gitignored)
├── run                      # Build & run script
├── release                  # Wrapper for scripts/release.sh
├── dist/                    # Release artifacts (gitignored)
└── docs/                    # Documentation
```

## Formatting

- Amounts display ≥2 decimal places
- Thousand separator is a space (e.g. `ZAR 12 345.67`)
- Currency is never assumed — always taken from the API response
