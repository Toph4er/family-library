# Family Library

A self-hosted web application for tracking and managing a child's book collection.

## Features

- **Book collection management** — Add, edit, search, and filter books with comprehensive metadata (ISBN, authors, illustrators, reading levels, genres, themes, awards, Dewey Decimal, series, language, description)
- **ISBN lookup & import** — Look up books by ISBN via the Open Library API with cached results for repeat lookups
- **Barcode scanning** — Scan book barcodes with a camera to auto-populate ISBN lookup
- **Wishlist management** — Track desired books with purchase links (Amazon, ThriftBooks) and mark items as fulfilled
- **Reading log** — Record reading sessions with page ranges, reader name, and notes
- **HTMX-powered UI** — Dynamic, partial-page updates without a full SPA framework
- **Alpine.js** — Lightweight interactivity for modals, dropdowns, and form validation
- **Tailwind CSS** — Utility-first styling (pre-built, no CDN)
- **Admin & guest authentication** — Role-based access with bcrypt-hashed passwords and cookie sessions
- **Per-field guest visibility** — Control which book fields guests can see
- **Family member management** — Track who gifted books (name and relationship)
- **RESTful API** — Full JSON API under `/api/v1` for programmatic access
- **Security** — CSRF protection, rate limiting, HSTS, CSP with nonces, and per-IP request throttling

## Tech Stack

- **Backend:** Go 1.26 + chi router + SQLite
- **Frontend:** Go `html/template` + HTMX + Alpine.js + Tailwind CSS
- **Database:** SQLite (pure Go driver, no CGO) with [goose](https://github.com/pressly/goose/v3) migrations
- **Deployment:** Docker + nginx reverse proxy

## Project Structure

```
family-library/
├── cmd/library/            # Application entry point (server setup, config, migrations)
├── internal/               # Private application packages
│   ├── api/                # HTTP router and route registration (chi)
│   ├── auth/               # Authentication (sessions, login, password hashing)
│   ├── db/                 # Database connection and initialization
│   ├── handlers/           # HTTP request handlers (books, auth, settings, wishlist, etc.)
│   ├── middleware/         # HTTP middleware (security headers, CSRF, rate limiting, logging)
│   ├── models/             # Data models (Book, WishlistItem, User, ReadingLog, etc.)
│   ├── repository/         # Database repository layer
│   ├── services/           # Business logic services
│   ├── theme/              # Theme system (themes, CSS overrides)
│   └── web/                # HTML templates, CSS, and static assets
│       ├── partials/       # Reusable template fragments (book cards, pagination, modals, etc.)
│       └── styles.css      # Tailwind source CSS (compiled to tailwind.css by build)
├── migrations/             # SQL migration files (14 migrations, applied with goose)
├── tests/                  # Integration tests (auth, books, settings, wishlist, HTML handlers)
├── .dockerignore           # Docker build exclusions
├── Dockerfile              # Multi-stage build (CSS builder → Go binary → Alpine runtime)
├── compose.yaml            # Docker Compose configuration
├── .gitlab-ci.yml          # GitLab CI/CD pipeline (optional — see below)
├── go.mod                  # Go module dependencies
├── package.json            # Node dependencies (Tailwind CSS build)
├── tailwind.config.js      # Tailwind CSS configuration
├── .env.example            # Environment variable reference
└── README.md               # This file
```

## How to Build and Run

### Prerequisites

- Go 1.26+
- Node.js 20+ (for Tailwind CSS build)
- Docker + Docker Compose

### Development

```bash
# Build the CSS (required before running)
npm run build:css

# Run the server (migrations auto-applied with --migrate flag)
go run ./cmd/library/ --migrate
```

The server listens on port `8080` by default. Set `DATABASE_PATH` to change the SQLite file location.

### Docker

```bash
# 1. Copy and customize the environment file
cp .env.example .env
# Edit .env with your own values

# 2. Build and run with Docker Compose
docker compose up -d
```

The Dockerfile uses a three-stage build:
1. **CSS builder** — installs Node.js dependencies and compiles Tailwind CSS
2. **Go builder** — downloads Go modules and compiles a static binary
3. **Runtime** — minimal Alpine image with the compiled binary, templates, and migrations

### Environment Variables

| Variable | Required | Description |
|---|---|---|
| `SESSION_SECRET` | Yes | Secret key for session signing (32+ random chars) |
| `ADMIN_USERNAME` | No | Initial admin username (seeded on first run) |
| `ADMIN_PASSWORD` | No | Initial admin password (seeded on first run) |
| `GUEST_PASSWORD` | No | Shared guest password (stored hashed in settings) |
| `PORT` | No | Server port (default: `8080`) |
| `DATABASE_PATH` | No | SQLite database path (default: `/app/data/library.db`) |
| `LOG_LEVEL` | No | Logging level: `debug`, `info`, `warn`, `error` (default: `info`) |
| `CORS_ORIGIN` | No | Allowed CORS origin (default: `https://example.com`) |
| `ENV` | No | Environment name; set to `development` to allow insecure cookies |
| `OL_BASE_URL` | No | Open Library API base URL (default: `https://openlibrary.org`) |
| `OL_COVERS_URL` | No | Open Library cover images URL (default: `https://covers.openlibrary.org`) |
| `OL_USER_AGENT` | No | User-Agent for Open Library requests (must identify your app) |
| `OL_HTTP_TIMEOUT` | No | HTTP timeout for OL requests (default: `10s`) |
| `OL_CACHE_TTL` | No | ISBN cache TTL (default: `24h`) |
| `OL_RATE_LIMIT_PER_SEC` | No | Rate limit for OL requests (default: `2`) |
| `VIRTUAL_HOST` | No | Reverse proxy hostname (default: `example.com`) |
| `LETSENCRYPT_HOST` | No | Let's Encrypt hostname (default: `example.com`) |

## Open Source

This project is designed to be self-hosted by anyone. To get started:

```bash
# Clone the repository
git clone https://github.com/Toph4er/family-library.git
cd family-library

# Copy and customize the environment file
cp .env.example .env
# Edit .env with your own SESSION_SECRET, ADMIN_PASSWORD, etc.

# Build the CSS
npm run build:css

# Start with Docker Compose
docker compose up -d
```

No CI/CD pipeline is required for self-hosting. The included `.gitlab-ci.yml` is only for the author's automated deployment.

## Reverse Proxy & SSL

This project is designed to run behind [nginx-proxy](https://github.com/nginx-proxy/nginx-proxy) and [acme-companion](https://github.com/nginx-proxy/acme-companion) for automatic reverse proxying and SSL certificate management.

### Requirements

1. **nginx-proxy** and **acme-companion** must be running on the same Docker network (named `reverse-proxy` by default)
2. The `reverse-proxy` network must be created as an **external** network (the compose file references `external: true`)

### Setup

```bash
# Create the shared network (if it doesn't exist)
docker network create reverse-proxy

# Start nginx-proxy
docker run -d -p 80:80 -p 443:443 \
  -v /var/run/docker.sock:/tmp/docker.sock:ro \
  --name nginx-proxy nginxproxy/nginx-proxy

# Start acme-companion
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v acme-data:/etc/acme.sh \
  -e NGINX_PROXY_CONTAINER=nginx-proxy \
  --name nginx-proxy-acme nginxproxy/acme-companion
```

### Configuration

Set these environment variables in your `.env` file:

| Variable | Description |
|---|---|
| `VIRTUAL_HOST` | Your domain (e.g., `library.example.com`) — nginx-proxy uses this to route traffic |
| `LETSENCRYPT_HOST` | Same domain — acme-companion uses this to request an SSL certificate |

Both must match your actual domain. The container will automatically obtain and renew a Let's Encrypt certificate.

### Alternative

You can use any reverse proxy (Traefik, Caddy, manual nginx, etc.) instead. Just ensure:

- HTTPS is terminated at the proxy
- The proxy forwards `Host` and `X-Forwarded-Proto` headers correctly
- The `CORS_ORIGIN` variable matches your domain
- The `ENV` variable is set to `production` (enables Secure cookie flag)

## CI/CD Pipeline

A GitLab CI pipeline is included (`.gitlab-ci.yml`) for automated testing, building, and deployment. It is **not required** for self-hosting — see the Docker instructions above.

1. **Test** — `go vet` + `go test` with coverage (15% minimum threshold)
2. **Build** — Multi-stage Docker build and push to registry
3. **Security** — Trivy filesystem scan, `gosec` static analysis, `golangci-lint`
4. **Deploy** — Pull latest image, start container, health check, and automatic rollback on failure

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

Built with 🌲🍄✨ for our little reader.
