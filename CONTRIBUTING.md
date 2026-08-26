# Contributing

A self-hosted Go + HTMX app for tracking and managing a child's book collection.

## Maintainer

This project is maintained by [@Toph4er](https://github.com/Toph4er). If you'd like to
contribute, **please open an issue first** describing what you want to change and why —
it's a small project and we don't want to work at cross purposes. Once there's an
agreement on the approach, a PR is welcome.

## Development setup

- Go 1.26+
- Node.js 20+ (only needed to rebuild the Tailwind CSS bundle)

```bash
git clone https://github.com/Toph4er/family-library.git
cd family-library

# Build
go build ./...

# Test (unit + integration)
go test ./...

# Lint (golangci-lint v2, config in .golangci.yml)
golangci-lint run ./...
```

## Code style

- Follow standard Go idioms (see `gofmt`/`goimports` via the linter).
- Error strings are **lowercase** and do not end with punctuation
  (enforced by `staticcheck` ST1005 — e.g. `fmt.Errorf("loading book: %w", err)`).
- Keep the repository layer free of HTTP concerns; handlers stay thin.

## Proposing changes

1. Open an issue describing the problem and your proposed approach.
2. Fork, branch, and make the smallest change that solves the problem.
3. Keep `golangci-lint run ./...`, `go vet ./...`, and `go test ./...` green.
4. Reference the issue in your PR and include a short summary of what changed and why.

## License

MIT — see [LICENSE](LICENSE).
