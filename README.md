# English Learning Bot

A comprehensive English learning platform consisting of a Telegram bot, REST API, and web interface to help users learn and practice English vocabulary through spaced repetition.

## Architecture Overview

The project consists of three main components:

1. **Backend API** (`cmd/api/`) - REST API for managing word translations and user data
2. **Web UI** (`web/`) - React-based web interface for vocabulary management
3. **Telegram Bot** (`cmd/bot/`) - Interactive bot for vocabulary practice and testing

## Features

### Telegram Bot
- **Word Practice**: Automated scheduled word checks with spaced repetition
- **Interactive Learning**: Users can mark words as guessed, missed, or for review
- **Statistics**: Track learning progress with detailed stats
- **Commands**:
  - `/start` - Get started with the bot
  - `/stats` - View learning statistics
  - `/random` - Get a random word to practice

### Web Interface
- **Word Management**: Create, edit, and delete word translations
- **Learning Progress**: Visual indicators for learned words (streak at or above the configured limit)
- **Filtering**: Filter by learning status (all, learned, batched, to_learn)
- **Search**: Find specific words and translations
- **Statistics Dashboard**: Charts and metrics showing learning progress
- **Keyboard shortcuts**: `q` to add word, `/` to focus search, `Escape` to blur

### API Features
- **Authentication**: JWT-based auth with Telegram chat ID verification
- **CRUD Operations**: Full word translation management
- **Statistics**: Comprehensive learning analytics
- **Rate Limiting**: Built-in protection against abuse
- **Security**: CORS, CSRF protection, secure headers

## Tech Stack

### Backend
- **Language**: Go 1.26.5
- **Web Framework**: Echo v4
- **Database**: SQLite
- **Telegram**: gopkg.in/telebot.v3
- **Authentication**: JWT tokens with HTTP-only cookies
- **Database Layer**: Squirrel query builder

### Frontend
- **Framework**: React 19.1.1
- **Build Tool**: Vite
- **UI Library**: React Bootstrap
- **Charts**: Chart.js with react-chartjs-2
- **Icons**: React Bootstrap Icons
- **Date Handling**: date-fns

## Learning Algorithm

The bot uses a spaced repetition system:
- Words start with a `guessed_streak` of 0
- Correct answers increment the streak
- Wrong answers reset the streak to 0
- Words reaching `BOT_LEARNING_STREAK_LIMIT` (default 15) are considered "learned" and leave the
  active batch
- The learning batch is topped back up to `BOT_LEARNING_BATCH_SIZE` (default 50) every hour

### Reviewing learned words

Learning a word once is not the same as remembering it. `BOT_LEARNING_REVIEW_RATE_PERCENT` of
scheduled checks (default 20%) re-test an already learned word instead of one from the active batch.
Review messages are prefixed with 🔁.

Reviews rotate: the least recently reviewed learned word is always picked next, so every learned word
comes up once before any of them comes up twice. The rotation advances when the message is **sent**,
so ignoring one does not stall it.

Answering ❌ on any word — review or not — resets its streak **and** puts it straight back into the
learning batch. That can push the batch above `BOT_LEARNING_BATCH_SIZE`; the hourly refill simply adds
nothing until words graduate out again.

Set `BOT_LEARNING_REVIEW_RATE_PERCENT=0` to disable reviews.

## Project Structure

```
├── cmd/                    # Application entry points
│   ├── api/               # REST API server
│   ├── bot/               # Telegram bot
│   └── import/            # Data import utility
├── internal/              # Internal packages
│   ├── api/              # API handlers and middleware
│   ├── config/           # Configuration management
│   ├── dal/              # Data access layer
│   ├── schedule/         # Background job scheduling
│   └── telegram/         # Telegram bot logic
├── web/                  # React frontend
│   ├── src/
│   │   ├── api/         # API client
│   │   ├── components/  # Reusable UI components
│   │   ├── routes/      # Page components
│   │   └── context.tsx  # App state management
├── schema/              # Database schemas
├── data/               # Database files
└── package/            # Built packages
```

## Database Schema

### Tables
- `word_translations` - Core vocabulary data with learning progress
- `learning_batches` - Words currently in active learning rotation
- `statistics` - Daily learning statistics per user
- `auth_confirmations` - Temporary authentication tokens
- `callback_data` - Telegram callback data storage

### Key Features
- Per-user data isolation using `chat_id`
- Automatic timestamp management with triggers
- Foreign key constraints for data integrity
- Indexes for optimal query performance

## Setup and Installation

### Prerequisites
- Go 1.26.5+
- Node.js 18+
- Telegram Bot Token
- SQLite

### Configuration

Create a `.env` file with the following variables:

```env
# Bot Configuration
BOT_TELEGRAM_TOKEN=your_telegram_bot_token
BOT_ALLOWED_CHAT_IDS=123456789,987654321
# Keep the busy_timeout pragma: without it a write that overlaps the hourly batch refill fails
# straight away with "database is locked" instead of waiting for it.
BOT_DB_PATH=file:data/db.sqlite?cache=shared&mode=rwc&_pragma=busy_timeout(5000)
BOT_DEV=false

# Schedule Configuration  
BOT_SCHEDULE_PUBLISH_INTERVAL=30m
BOT_SCHEDULE_HOUR_FROM=9
BOT_SCHEDULE_HOUR_TO=22
BOT_SCHEDULE_TIMEZONE=Europe/London

# Learning Configuration
BOT_LEARNING_BATCH_SIZE=50
BOT_LEARNING_STREAK_LIMIT=15
BOT_LEARNING_REVIEW_RATE_PERCENT=20

# API Configuration
API_TELEGRAM_TOKEN=your_telegram_bot_token
API_TELEGRAM_ALLOWED_CHAT_IDS=123456789,987654321
API_DB_PATH=./data/db.sqlite
API_DEV=false
API_SERVER_ADDR=:8080
API_SERVER_READ_HEADER_TIMEOUT=30s

# HTTP Configuration
API_HTTP_RATE_LIMIT=100
API_HTTP_PROCESS_TIMEOUT=30s
API_HTTP_JWT_SECRET=your_jwt_secret
API_HTTP_JWT_EXPIRY=24h
API_HTTP_COOKIE_AUTH_EXPIRES_IN=24h
API_HTTP_COOKIE_ACCESS_EXPIRES_IN=15m
API_HTTP_CORS_ALLOW_ORIGINS=http://localhost:3000,https://yourdomain.com

# Web Configuration
VITE_API_BASE_URL=http://localhost:8080
```

### Build and Run

1. **Initialize database**:
   ```bash
   sqlite3 data/db.sqlite < schema/schema_sqlite.sql
   ```

   For an **existing** database, apply any migrations it has not seen yet — they are additive and
   safe to run against live data, but each one only once:
   ```bash
   sqlite3 data/db.sqlite < schema/migrations/001_last_reviewed_seq.sql
   ```

2. **Build the applications**:
   ```bash
   make build
   ```
3. **Start the API server**:
   ```bash
   ./bin/english-learning-api
   # or
   ./run-api.sh
   ```

4. **Start the Telegram bot**:
   ```bash
   ./bin/english-learning-bot  
   # or
   ./run-bot.sh
   ```

5. **Start the web interface**:
   ```bash
   cd web
   npm install
   npm run dev
   ```

## Authentication Flow

1. User visits web interface
2. Login page prompts for Telegram Chat ID
3. System sends confirmation message to Telegram
4. User confirms in Telegram bot
5. Web interface receives JWT token
6. Subsequent requests use HTTP-only cookies

## API Endpoints

### Authentication
- `POST /auth/login` - Initiate login process
- `GET /auth/status` - Check authentication status
- `GET /auth/info` - Get user information
- `POST /auth/logout` - Logout user

### Words Management
- `GET /words` - List words with filtering and pagination
- `POST /words` - Create new word translation. If the word already exists, responds `409` with the
  stored entry instead of overwriting it; resend with `"on_conflict"` set to `reset_and_batch`,
  `reset_only` or `update_only` to apply a decision
- `PUT /words` - Update existing word translation
- `PUT /words/review` - Mark word for review
- `POST /words/reset` - Reset a word's streak to 0, optionally putting it back into the learning
  batch (`{"word": "...", "add_to_batch": true}`)
- `DELETE /words` - Delete word translation

### Statistics
- `GET /stats/total` - Get overall learning statistics
- `GET /stats` - Get daily statistics
- `GET /stats/range` - Get statistics for date range

### Health
- `GET /health` - Unauthenticated. Returns `{"status", "version", "build_time"}` of the running
  backend. See [Build version](#build-version)

## Build version

The running build is identifiable three ways: the `starting` log line, `GET /health`, and a
tooltip on the "Home" brand link in the web UI navbar (which reads it from `/health`, so it
always reports the backend that is actually serving the session).

`Version` and `BuildTime` are stamped into `cmd/bot/main.go` via `-ldflags` at compile time:

- **Local builds** (`make build-bot`) stamp `dev` — `VERSION ?= dev` in the `Makefile`.
- **CI builds** stamp the short commit SHA, which `.github/workflows/docker.yml` passes as the
  `VERSION` Docker build-arg. The bot image is tagged with the same SHA.

To stamp a local build explicitly:

```bash
VERSION=$(git rev-parse --short HEAD) make build-bot
```

Note the web image is versioned by its GHCR tag only; it carries no build-time version of its
own, since the navbar reports the backend's.

## Development

### Running Tests
```bash
go test ./...
```

### Code Quality
```bash
# Lint Go code
golangci-lint run

# Lint TypeScript/React
cd web && npm run lint
```

### Database Migrations
The application uses SQL schema files for database setup. For schema changes:

1. Update `schema/schema_sqlite.sql`
2. Apply changes to your development database
3. Test with SQLite if needed

## Deployment

The project uses Docker for deployment. Images are built and pushed to GHCR automatically on push to `main`.

### Docker Compose (local development)
```bash
BOT_TELEGRAM_TOKEN=<token> BOT_TELEGRAM_ALLOWED_CHAT_IDS=<ids> make docker-up
```

### Production
Use `docker-compose.prod.yml` with pre-built images from GHCR:
```bash
docker compose -f docker-compose.prod.yml up -d
```

### CI/CD Workflow
1. Push to `main` branch
2. GitHub Actions builds and pushes multi-arch Docker images (amd64/arm64) to GHCR
3. Pull latest images on the server

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run linters and tests
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built with [Echo](https://echo.labstack.com/) web framework
- Telegram integration via [telebot](https://gopkg.in/telebot.v3)
- Frontend powered by [React](https://react.dev/) and [Bootstrap](https://getbootstrap.com/)
