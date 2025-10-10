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
- **Learning Progress**: Visual indicators for learned words (15+ streak)
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
- **Language**: Go 1.24.1
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
- Words with 15+ streaks are considered "learned"
- Learning batches automatically update based on performance

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
- Go 1.24+
- Node.js 18+
- Telegram Bot Token
- SQLite

### Configuration

Create a `.env` file with the following variables:

```env
# Bot Configuration
BOT_TELEGRAM_TOKEN=your_telegram_bot_token
BOT_ALLOWED_CHAT_IDS=123456789,987654321
BOT_DB_URL=./data/db.sqlite
BOT_DEV=false

# Schedule Configuration  
BOT_SCHEDULE_PUBLISH_INTERVAL=30m
BOT_SCHEDULE_HOUR_FROM=9
BOT_SCHEDULE_HOUR_TO=22
BOT_SCHEDULE_TIMEZONE=Europe/London

# API Configuration
API_TELEGRAM_TOKEN=your_telegram_bot_token
API_TELEGRAM_ALLOWED_CHAT_IDS=123456789,987654321
API_DB_URL=./data/db.sqlite
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
- `POST /words` - Create new word translation
- `PUT /words` - Update existing word translation
- `PUT /words/review` - Mark word for review
- `DELETE /words` - Delete word translation

### Statistics
- `GET /stats/total` - Get overall learning statistics
- `GET /stats` - Get daily statistics
- `GET /stats/range` - Get statistics for date range

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

This project includes automated CI/CD for AWS EC2 deployment. See [`deployment/README.md`](deployment/README.md) for complete setup instructions.

### Quick Start (EC2/Amazon Linux 2)
```bash
# One-command setup
curl -sfL https://raw.githubusercontent.com/Roma7-7-7/english-learning-bot/main/deployment/setup-ec2.sh | sudo bash
```

This sets up:
- Systemd services for API and bot
- Automatic hourly deployment from GitHub releases
- Health monitoring and auto-restart
- Shared SQLite database

### CI/CD Workflow
1. Push to `main` branch
2. GitHub Actions builds binaries (with version info)
3. Creates GitHub Release
4. EC2 automatically deploys within an hour

### Manual Deployment
For non-EC2 or custom deployments:
1. Build binaries: `make build-release VERSION=v1.0.0`
2. Build web assets: `cd web && npm run build`
3. Deploy binaries and web assets to server
4. Configure environment variables
5. Set up systemd services (see `deployment/systemd/`)

See also:
- [`deployment/README.md`](deployment/README.md) - Complete deployment guide
- [`deployment/QUICKREF.md`](deployment/QUICKREF.md) - Common operations
- [`deployment/MAKEFILE.md`](deployment/MAKEFILE.md) - Build system documentation

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