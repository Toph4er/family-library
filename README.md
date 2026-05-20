# Library Book Collection

A self-hosted web application for tracking and managing a child's book collection. This version represents a major architectural modernization, transitioning from complex client-side frameworks (like React) to a robust, performant, and delightful server-driven architecture powered by HTMX.

## Architectural Shift: Embracing Server-Driven UI with HTMX

We are thrilled to announce the completion of our migration from client-side rendering frameworks (React) to a pure, server-driven architecture using **HTMX**. This modernization effort significantly improves development speed, reduces complexity, and enhances performance by keeping the entire application logic within the Go backend.

**Why this change?**
*   **Simplicity & Reliability:** By eliminating large JavaScript build pipelines and complex state management layers (common in React setups), we have drastically simplified our stack. The whole system runs on Go templates and HTTP requests, making it easier to maintain and debug.
*   **Performance Focus:** HTMX allows us to achieve dynamic, modern user experiences—like partial page updates, AJAX forms, and interactive components—without writing a single line of complex JavaScript state logic. We get the best of both worlds: the power of client-side interactivity with the reliability of server-side rendering.
*   **Developer Experience (DX):** The entire stack is now Go-centric. This means fewer context switches for developers and faster iteration cycles, allowing us to focus purely on book metadata and user experience rather than framework plumbing.

This transition allows us to deliver a highly responsive application while maintaining the simplicity and robustness that Go excels at.

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
- **Frontend:** Go html/templates + HTMX + Alpine.js
- **Database:** SQLite (pure Go driver, no CGO)
- **Deployment:** Docker + GitLab CI/CD + nginx reverse proxy

## Quick Start

### Prerequisites
- Go 1.26+
- Docker + Docker Compose

### Development
The entire application is now served by the Go backend. No separate Node.js environment or frontend build step is required for local development.

```bash
# Backend (run from project root)
go run .
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
| :--- | :--- |
| `ADMIN_USERNAME` | Initial admin username |
| `ADMIN_PASSWORD` | Initial admin password |
| `SESSION_SECRET` | Secret key for session signing (32+ random chars) |
| `GUEST_PASSWORD` | Shared guest password |
| `LOG_LEVEL` | Logging level (debug/info/warn/error) |

### Optional CI/CD Variables

| Variable | Description |
| :--- | :--- |
| `LITESTREAM_AWS_ACCESS_KEY_ID` | S3 access key for Litestream backups |
| `LITESTREAM_AWS_SECRET_ACCESS_KEY` | S3 secret key for Litestream backups |
| `LITESTREAM_REPOSITORY` | S3 bucket/path for backups |

## Project Structure

```
library/
├── backend/          # Go API server logic
├── internal/         # Private application packages (e.g., models, services)
│   └── templates/    # Templates directory used by the Go renderer
├── Dockerfile        # Multi-stage build
├── compose.yaml      # Docker Compose config
├── .gitlab-ci.yml    # CI/CD pipeline
└── design-docs/      # Local design docs (gitignored)
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

Built with 🌲🍄✨ for our little reader.