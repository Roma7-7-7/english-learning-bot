# Claude Development Context

This document provides context for AI assistants working on this English Learning Bot project.

## Project Overview

This is a personal English learning platform with three main components:
1. **Backend API** (Go) - REST API for vocabulary management
2. **Web UI** (React/TypeScript) - Web interface for managing translations
3. **Telegram Bot** (Go) - Interactive bot for vocabulary practice

## Architecture & Design Patterns

### Backend (Go)
- **Framework**: Echo v4 web framework
- **Architecture**: Clean architecture with separate layers
- **Database**: SQLite
- **Authentication**: JWT with HTTP-only cookies + Telegram verification
- **Error Handling**: Structured logging with slog
- **Middleware**: Rate limiting, CORS, security headers, request logging

### Frontend (React/TypeScript)
- **Framework**: React 19.1.1 with TypeScript
- **Build Tool**: Vite
- **UI**: React Bootstrap components
- **State Management**: React Context API
- **Authentication**: JWT stored in HTTP-only cookies
- **Charts**: Chart.js for statistics visualization

### Database Design
- **Per-user isolation**: All tables use `chat_id` for user separation
- **Learning algorithm**: `guessed_streak` field tracks spaced repetition progress
- **Batch system**: `learning_batches` table manages active learning words
- **Statistics**: Daily tracking in `statistics` table

## Key Business Logic

### Learning Algorithm
- Words start with `guessed_streak = 0`
- Correct answers: increment streak
- Wrong answers: reset streak to 0
- Words with 15+ streaks considered "learned"
- Batch system rotates active learning words

### Authentication Flow
1. User enters Telegram chat ID in web UI
2. API sends confirmation message via Telegram bot
3. User confirms in Telegram
4. Web UI receives JWT token
5. Subsequent requests use HTTP-only cookies

### Scheduling System
- Background goroutine sends word checks at configured intervals
- Time-based filtering (only during specified hours)
- Per-chat scheduling with error handling
- Graceful shutdown handling

## Claude Code Configuration

This project includes custom slash commands and skills for Claude Code:

**Slash Commands:**
- `/commit` - Analyze staged changes and create a commit with an appropriate message
- `/prep-pr` - Prepare current PR for review by verifying completeness and updating documentation

**Skills:**
- `shell-scripts` - Shell script development guidelines and best practices
- `sonarqube` - SonarCloud code quality analysis and compliance (all languages)

See `.claude/commands/` and `.claude/skills/` for implementation details.

## Development Environment

### Go Configuration
- Version: 1.25.4
- Key dependencies:
  - `github.com/labstack/echo/v4` - Web framework
  - `gopkg.in/telebot.v3` - Telegram bot
  - `github.com/Masterminds/squirrel` - SQL query builder
  - `github.com/golang-jwt/jwt/v5` - JWT tokens

### Frontend Configuration
- Node.js with TypeScript
- Key dependencies:
  - `react@19.1.1` - UI framework
  - `react-bootstrap` - UI components
  - `chart.js` - Data visualization
  - `react-router-dom` - Routing

### Build Commands

**Use Makefile targets** (single source of truth for CI and local builds):

```bash
# Local development (native OS/arch)
make build-api          # Build API
make build-bot          # Build bot
make build-backend      # Build both
make build-web          # Build frontend
make build              # Build everything

# Production builds (Linux ARM 64 - used by CI)
make build-release      # Build both binaries (requires Linux or Docker)

# With version info
make build-api VERSION=v1.0.0 BUILD_TIME=$(date -u +%Y%m%d-%H%M%S)

# See all available targets
make help
```

**Note**: Production builds with `make build-release` require Linux due to CGO (go-sqlite3). On macOS, use `make build-backend` for local development. Let GitHub Actions handle production builds.

### Testing Commands
```bash
# Go tests
make test               # Run tests
make vet                # Run go vet
make ci-test            # Run both (used by CI)

# Frontend linting
cd web && npm run lint

# Go linting (optional, requires golangci-lint)
make lint
```

### Code Quality - SonarCloud

This project uses **SonarCloud** for continuous code quality analysis. See `.claude/skills/sonarqube.md` for:
- How to check issues via API or Web UI
- Common SonarQube rules for Go, TypeScript, and Shell
- How to fix compliance issues
- CI/CD integration details

**Quick check:**
```bash
curl -s "https://sonarcloud.io/api/issues/search?componentKeys=Roma7-7-7_english-learning-bot&statuses=OPEN,CONFIRMED&sinceLeakPeriod=true&ps=100" | \
  jq -r '.issues[] | "\(.rule) | \(.component) | Line \(.line // "N/A") | \(.message)"'
```

## File Organization

### Key Backend Files
- `cmd/api/main.go` - API server entry point
- `cmd/bot/main.go` - Telegram bot entry point
- `internal/api/routes.go` - API route definitions and middleware
- `internal/api/words.go` - Word management handlers
- `internal/api/auth.go` - Authentication handlers
- `internal/telegram/bot.go` - Telegram bot logic
- `internal/dal/` - Database access layer
- `internal/config/` - Configuration management

### Key Frontend Files
- `web/src/App.tsx` - Main application component
- `web/src/routes/Home.tsx` - Word management interface
- `web/src/routes/Stats.tsx` - Statistics dashboard
- `web/src/api/client.tsx` - API client implementation
- `web/src/components/` - Reusable UI components

### Database Files
- `schema/schema_sqlite.sql` - SQLite schema
- `data/` - Database files (SQLite)

## Configuration

### Environment Variables
The application uses environment-based configuration with prefixes:
- `BOT_*` - Telegram bot configuration
- `API_*` - API server configuration
- `VITE_*` - Frontend build configuration

### Configuration Sources

Both Bot and API configurations support two modes:

1. **Environment Variables** (recommended for simple deployments):
   - Set all required variables in `.env` file or environment
   - Skips AWS SSM lookup entirely
   - Works on any server without AWS dependencies

2. **AWS SSM Parameter Store** (production EC2 deployments):
   - If required env vars are not set and `DEV=false`
   - Fetches secrets from SSM parameters
   - Requires IAM permissions for `ssm:GetParameters`

**Bot Required Parameters** (via env or SSM):
- `BOT_TELEGRAM_TOKEN` - Telegram bot token
- `BOT_ALLOWED_CHAT_IDS` - Comma-separated allowed chat IDs

**API Required Parameters** (via env or SSM):
- `API_HTTP_JWT_SECRET` - JWT signing secret
- `API_TELEGRAM_TOKEN` - Telegram bot token (for sending notifications)
- `API_TELEGRAM_ALLOWED_CHAT_IDS` - Comma-separated allowed chat IDs

### Key Configuration Areas
1. **Database**: Connection strings, timeouts
2. **Telegram**: Bot token, allowed chat IDs
3. **Scheduling**: Intervals, time windows, timezone
4. **HTTP**: CORS, rate limiting, timeouts
5. **Security**: JWT secrets, cookie settings

## Common Development Tasks

### Adding New API Endpoints
1. Add handler to appropriate file in `internal/api/`
2. Register route in `internal/api/routes.go`
3. Add corresponding client method in `web/src/api/client.tsx`
4. Update TypeScript interfaces if needed

### Database Changes
1. Update schema files in `schema/`
2. Add corresponding Go structs in `internal/dal/models.go`
3. Update repository interfaces and implementations
4. Test with SQLite if applicable

### Frontend Components
- Follow existing patterns in `web/src/components/`
- Use React Bootstrap for consistency
- Implement proper TypeScript interfaces
- Handle loading and error states

### Telegram Bot Features
1. Add command handlers in `internal/telegram/bot.go`
2. Register handlers in the `Start()` method
3. Update callback handling if needed
4. Test with actual Telegram bot

## Security Considerations

### Authentication
- JWT tokens with short expiry (15m access, 24h refresh)
- HTTP-only cookies prevent XSS
- CSRF protection via custom headers
- Rate limiting per IP address

### Data Protection
- Chat ID-based data isolation
- No sensitive data logging
- Secure headers (HSTS, CSP, etc.)
- Input validation on all endpoints

### Telegram Integration
- Bot token stored as environment variable
- Allowed chat ID whitelist
- Callback data expiration (7 days)
- Error handling for blocked users

## Troubleshooting

### Common Issues
1. **Database locked**: SQLite concurrency issues - check for long-running transactions
2. **Authentication failures**: Check JWT secret, cookie settings, CORS configuration
3. **Telegram bot not responding**: Verify token, check logs, ensure proper error handling
4. **Frontend API errors**: Check CORS settings, verify API endpoints

### Debugging
- Use structured logging with slog in Go
- Browser dev tools for frontend debugging
- Monitor database connections and queries

## Performance Considerations

### Database
- Indexes on frequently queried columns (`chat_id`, `date`)
- Connection pooling for concurrent access
- Batch operations where possible
- Regular cleanup of expired data

### API
- Rate limiting prevents abuse
- Request timeouts prevent hanging
- Middleware for compression and caching
- Efficient query patterns

### Frontend
- Code splitting with Vite
- Lazy loading of components
- Optimized bundle size
- Efficient state updates

## Deployment Notes

### Deployment Options

The project supports two deployment modes:

1. **Simple Deployment** (recommended for most users)
   - Works on any Linux server (Hetzner, Contabo, OVH, etc.)
   - No AWS dependencies
   - Configuration via `.env` file
   - Manual backups (SCP from local machine)
   - Lower costs (~$5/month vs ~$15/month)
   - See `deployment/SIMPLE-DEPLOYMENT.md`

2. **AWS EC2 Deployment**
   - Automated S3 backups
   - AWS SSM Parameter Store for secrets
   - IAM role-based authentication
   - See `deployment/README.md` (EC2 section)

### Multi-Architecture Support

The build system produces binaries for both architectures:

- `english-learning-api-amd64` / `english-learning-bot-amd64` - For Intel/AMD x86_64 processors (most VPS providers)
- `english-learning-api-arm64` / `english-learning-bot-arm64` - For ARM64 processors (AWS Graviton, etc.)

The `deploy.sh` script automatically detects the server architecture using `uname -m` and downloads the correct binaries.

### Automated CI/CD

- **GitHub Actions**: Builds both AMD64 and ARM64 binaries on push to `main`, creates releases with version info
- **Deployment**: Systemd services with automatic updates (or manual via `deploy.sh`)
- **Version Tracking**: Build-time version injection via `-ldflags`, exposed in logs and `/health` endpoint
- **Documentation**: Complete setup guides in `deployment/` directory

**Key Files**:
- `.github/workflows/release.yml` - CI/CD workflow (multi-arch builds)
- `deployment/setup-simple.sh` - Simple deployment setup
- `deployment/setup-ec2.sh` - AWS EC2 setup
- `deployment/deploy.sh` - Automated deployment script (auto-detects architecture)
- `deployment/systemd/*-simple.service` - Simple deployment systemd services
- `deployment/systemd/*.service` - EC2 systemd services
- `Makefile` - Unified build system (builds both architectures for CI)

### Production Considerations
- Configure proper backup strategies (automated S3 or manual SCP)
- Monitor resource usage and performance
- Set up alerting for critical failures
- Use HTTPS in production environments
- See `deployment/README.md` or `deployment/SIMPLE-DEPLOYMENT.md` for complete documentation