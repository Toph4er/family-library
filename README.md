# Library Book Collection

A self-hosted web application for tracking and managing a child's book collection.

## Features

- Woodland fairy tale themed interface
- Admin and guest authentication
- Comprehensive book metadata (ISBN, authors, reading levels, genres, themes, awards)
- ISBN-based book import with cover image fetching (Google Books API → Open Library fallback)
- Wishlist management with purchase links (Amazon, ThriftBooks)
- Per-field guest visibility control
- Search and filter by author, type, level, tags, and more

## Tech Stack

- **Backend:** Go 1.26 + chi router + SQLite
- **Frontend:** React 19 + Vite 8 + Tailwind CSS 4
- **Database:** SQLite (pure Go driver, no CGO)
- **Deployment:** Docker + GitLab CI/CD + nginx reverse proxy

## Quick Start

### Prerequisites

- Go 1.26+
- Node.js 22+
- Docker + Docker Compose

### Development

```bash
# Backend
cd backend
go mod download
go run .

# Frontend (in another terminal)
cd frontend
npm install
npm run dev
```

### Docker

```bash
docker compose up -d
```

## Deployment

1. Create a project at `git.rcsmaine.com/chris/library`
2. Set CI/CD variables (see below)
3. Push to `main` branch
4. Pipeline runs: test → build → deploy

### Required CI/CD Variables

| Variable | Description |
|----------|-------------|
| `ADMIN_USERNAME` | Initial admin username |
| `ADMIN_PASSWORD` | Initial admin password |
| `SESSION_SECRET` | Secret key for session signing (32+ random chars) |
| `GUEST_PASSWORD` | Shared guest password |
| `LOG_LEVEL` | Logging level (debug/info/warn/error) |

### Optional CI/CD Variables

| Variable | Description |
|----------|-------------|
| `LITESTREAM_AWS_ACCESS_KEY_ID` | S3 access key for Litestream backups |
| `LITESTREAM_AWS_SECRET_ACCESS_KEY` | S3 secret key for Litestream backups |
| `LITESTREAM_REPOSITORY` | S3 bucket/path for backups |

## Project Structure

```
library/
├── backend/          # Go API server
├── frontend/         # React SPA
├── Dockerfile        # Multi-stage build
├── compose.yaml      # Docker Compose config
├── .gitlab-ci.yml    # CI/CD pipeline
└── design-docs/      # Local design docs (gitignored)
```

## License

TBD

---

Built with 🌲🍄✨ for our little reader.
