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
- **Database**: SQLite primary, PostgreSQL support via interface
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

## Development Environment

### Go Configuration
- Version: 1.24.1
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
```bash
# Backend
go build -o bin/english-learning-api ./cmd/api
go build -o bin/english-learning-bot ./cmd/bot

# Frontend
cd web && npm run build

# All at once
make build
```

### Testing Commands
```bash
# Go tests
go test ./...

# Frontend linting
cd web && npm run lint

# Go linting (if available)
golangci-lint run
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
- `schema/schema_postgre.sql` - PostgreSQL schema
- `data/` - Database files (SQLite)

## Configuration

### Environment Variables
The application uses environment-based configuration with prefixes:
- `BOT_*` - Telegram bot configuration
- `API_*` - API server configuration
- `VITE_*` - Frontend build configuration

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
4. Test with both SQLite and PostgreSQL if applicable

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
- Check Docker logs if using containers
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

### Docker
- Multi-stage builds for optimized images
- Environment variable configuration
- Volume mounting for database persistence
- Health checks for monitoring

### Production Considerations
- Use PostgreSQL instead of SQLite for better concurrency
- Configure proper backup strategies
- Monitor resource usage and performance
- Set up alerting for critical failures
- Use HTTPS in production environments