# Contributing

Thanks for taking an interest. The goal is that you can clone, change, and test in under 10 minutes.

## Setup

Requires [Go](https://go.dev/dl/) (see `go.mod` for the version).

```bash
git clone https://github.com/d4n13v4nt0nd3r/investec.tui.git
cd investec.tui
./run
```

With no credentials file, the app opens a setup screen. For a hand-written config:

```bash
cp env.example .env
```

Never commit `.env`, `investec.env`, or real Client ID / Client Secret / API Key values.

## Checks before a PR

```bash
go vet ./...
go test -race ./...
```

## Scope

- Prefer small, focused PRs.
- Do not add mock banking data unless it is temporary test fixtures with obvious fake values.
- Currency and amount formatting: never assume `$`; use a space as the thousands separator; show at least 2 decimal places.

## Releases

Maintainers publish Mac/Windows packages with `./release` as described in [docs/releasing.md](docs/releasing.md).
