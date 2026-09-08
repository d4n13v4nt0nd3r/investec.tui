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

   macOS will warn the first time you open it, because we do not yet pay for an Apple developer certificate. Step 3 shows how to get past it.

### Windows

1. Download `InvestecTUI-<version>-Windows-x64.exe`. Choose the `ARM64` file instead only if you have a Windows-on-ARM machine such as a Surface Pro X.
2. Move it somewhere permanent, such as your **Desktop** or **Documents** folder.

   Windows will warn you that the file is not commonly downloaded, because we do not yet pay for a Microsoft signing certificate. Choose **Keep** to complete the download.

## Step 3: Run the App

### macOS

Open your **Applications** folder, **Control-click** (or right-click) **InvestecTUI** and choose **Open**, then **Open** again in the dialog. A Terminal window opens and the app runs inside it.

The long way round is only needed the first time. macOS says it *"cannot verify the developer"* because the app is not signed by Apple yet; after that first Open, double-clicking works normally.

If you double-clicked by mistake and got a dead end, open **System Settings** -> **Privacy & Security**, scroll down, and click **Open Anyway** next to InvestecTUI.

### Windows

Double-click the `.exe` you saved. A console window opens and the app runs inside it.

The first time you run it, Windows shows a blue **"Windows protected your PC"** screen. This is expected for an unsigned app. Click **More info**, then **Run anyway**. Windows only asks once.

## Step 4: Enter Your Keys

The first run opens a setup screen that walks you through it. Nothing has to be created by hand.

1. Tick the countries you bank in.
2. Paste the Client ID, Client Secret and API Key for each one. They are hidden as you type; press `ctrl+r` to check a value you are unsure of.
3. The app tries the keys against Investec, then saves them.

They are saved, readable only by you, in:

| System | Where they are saved |
|--------|----------------------|
| macOS | `~/Library/Application Support/investec-tui/investec.env` |
| Windows | `%APPDATA%\investec-tui\investec.env` |

To change or add keys later, press `c` on the country page.

> **Important:** treat that file like a password. Do not email it around or store it in a shared folder.

### Writing the file yourself instead

If you would rather not use the setup screen, save a plain text file at the path above containing:

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
- On Windows you can also simply keep `investec.env` in the same folder as the `.exe`.

## Updating the App
Already downloaded the app before? Get the latest changes like this:
### macOS
```bash
cd tui.investec-openbanking.go
git pull
./run
```
`./run` always rebuilds the app before running it, so pulling the latest code is all you need.
### Windows (Git Bash)
```bash
cd tui.investec-openbanking.go
git pull
go build -o investec.openbanking.tui.exe .
./investec.openbanking.tui.exe
```
Notes:
- Your `.env` file is never touched by `git pull` — it's gitignored, so your credentials are safe.
- If a new setting was added, compare the template to your file and add any missing lines:
  ```bash
  diff env.example .env
  ```
- If `git pull` reports a conflict or says you have local changes, run `git status` to see what changed before doing anything else — you likely edited a tracked file by accident.
## Troubleshooting

| Problem | Solution |
|---------|----------|
| The setup screen says a key failed | Press `e` to go back and re-paste it. Investec shows the exact values under Programmable Banking -> Open API. |
| The setup screen could not save | The path it shows is not writable. Check the folder exists and belongs to you, or point `INVESTEC_TUI_ENV` at a file you can write. |
| `Authentication failed` | Press `c` on the country page and re-enter the keys for that country |
| A country shows `missing` on the landing page | Press `c` (or Enter on that country) and fill in its keys |
| App shows no accounts | Your API access may not be enrolled yet — revisit Step 1 |
| Windows: "Windows protected your PC" | Click **More info** -> **Run anyway**. Only needed once. |
| Windows: the window flashes and disappears | It should now pause and wait for Enter. If it does not, open PowerShell, drag the `.exe` into the window and press Enter to see the message. |
| macOS: "cannot verify the developer" or "Apple could not verify" | Expected: the app is not signed yet. Control-click it in **Applications** and choose **Open**, or use **Open Anyway** in **System Settings** -> **Privacy & Security**. |
| macOS: nothing happens on double-click | Make sure you dragged the app to **Applications** from the disk image rather than running it from the mounted image |

## For Developers: Build From Source

Requires [Go](https://go.dev/dl/) and Git.

```bash
git clone https://github.com/BeamMoney/tui.investec-openbanking.go.git
cd tui.investec-openbanking.go
./run
```

With no `.env` in the clone, `./run` opens the same setup screen a downloaded copy shows, and writes the answers to the repo's `.env`. Filling in `env.example` by hand still works:

```bash
cp env.example .env
$EDITOR .env
```

The app looks for a credentials file in this order, stopping at the first hit. In each folder it tries `investec.env` first, then `.env`:

1. The full path in the `INVESTEC_TUI_ENV` environment variable, if set
2. The current working directory
3. The folder holding the executable
4. The per-user config folder from the table in Step 4

So `./run` from a clone keeps using the repo's own `.env`, and a packaged build falls through to the per-user folder. The setup screen saves to whichever file is already in use, and to the per-user folder when there is none.

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
| c         | Change or add credentials |
| q         | Quit                |

### Setup Screen

| Key       | Action              |
|-----------|---------------------|
| Enter     | Next question       |
| Shift+Tab | Previous question   |
| Space     | Tick a country      |
| ctrl+r    | Show or hide the value being entered |
| ctrl+s    | Save without checking the keys first |
| Esc       | Back a step         |

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
| p     | View pending transactions |
| d     | View documents (statements / tax certificates) |
| r     | Refresh balance    |
| Esc   | Back to accounts   |

### Transactions View

| Key       | Action                      |
|-----------|-----------------------------|
| ↑/↓ or k/j | Scroll transactions       |
| f         | Filter by date range        |
| e         | Export loaded transactions to CSV |
| r         | Refresh with current filter |
| Esc       | Back to balance             |

When filtering dates, type in `YYYY-MM-DD` format. Press Enter to confirm each field (from → to), then transactions reload automatically.

### Documents View

| Key       | Action                      |
|-----------|-----------------------------|
| ↑/↓ or k/j | Scroll documents          |
| Enter     | Download selected PDF (save-as) |
| f         | Filter by date range        |
| r         | Refresh with current filter |
| Esc       | Back to balance             |

### Save As

Opened from document download or CSV export. Default folder is Downloads; default filename is the account number.

| Key       | Action                      |
|-----------|-----------------------------|
| Type      | Edit filename               |
| Tab       | Switch between filename and folder list |
| ↑/↓       | Browse folders              |
| Enter     | Save (filename focused) or open folder |
| y / n     | Confirm or cancel overwrite |
| Esc       | Cancel                      |

API keys need **View statements** / **View tax certificates** permissions for PDF downloads.

## API Endpoints Used

The country code selected on the landing page becomes the first path segment (`za`, `mu`, ...).

| Endpoint | Description |
|----------|-------------|
| `POST /identity/v2/oauth2/token` | OAuth2 client_credentials auth |
| `GET /{country}/pb/v1/accounts` | List accounts |
| `GET /{country}/pb/v1/accounts/{id}/balance` | Account balance |
| `GET /{country}/pb/v1/accounts/{id}/transactions` | Transaction history |
| `GET /{country}/pb/v1/accounts/{id}/documents` | List statements / tax certificates |
| `GET /{country}/pb/v1/accounts/{id}/document/{type}/{date}` | Download a PDF document |

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
│   │   ├── env_file.go      # Where the credentials file is looked for
│   │   └── save.go          # Writing the credentials file back out
│   ├── export/
│   │   └── export.go        # CSV encoding and file path helpers
│   ├── startup/
│   │   ├── console.go       # Terminal detection, press-Enter pause
│   │   ├── relaunch_*.go    # macOS: hand a double-clicked .app to Terminal
│   │   └── setup.go         # Printed fallback when there is no terminal for the setup screen
│   └── tui/
│       ├── app.go           # Bubble Tea model, routing
│       ├── setup.go         # Guided, masked credentials entry
│       ├── country.go       # Country selection landing page
│       ├── accounts.go      # Accounts list view
│       ├── balance.go       # Balance detail view
│       ├── transactions.go  # Transactions table view
│       ├── documents.go     # Statement / tax certificate list
│       ├── saveas.go        # Save-as filename + folder picker
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
