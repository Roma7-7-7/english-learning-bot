# Makefile Documentation

This document explains the Makefile structure and how it unifies build commands for both local development and CI/CD.

## Philosophy

The Makefile is the **single source of truth** for all build commands. GitHub Actions and local development use the same Makefile targets, ensuring consistency and eliminating duplication.

## Quick Reference

```bash
# Show all available targets
make help

# Local development
make build-api           # Build API for your OS/arch
make build-bot           # Build bot for your OS/arch
make build-backend       # Build both API and bot

# Testing
make test                # Run tests
make vet                 # Run go vet
make lint                # Run golangci-lint (if installed)

# CI/CD (used by GitHub Actions)
make ci-test             # Run all CI tests
make ci-build            # Build production binaries

# Custom version
make build-api VERSION=v1.2.3 BUILD_TIME=20250109-103000

# Info
make info                # Show build configuration
```

## Target Categories

### 1. Development Builds

**`make build-api-local`** / **`make build-api`**
- Builds API for your native OS/architecture
- Includes version info (defaults to "dev")
- Uses CGO (required for SQLite)

**`make build-bot-local`** / **`make build-bot`**
- Builds bot for your native OS/architecture
- Includes version info (defaults to "dev")
- Uses CGO (required for SQLite)

**`make build-backend`**
- Builds both API and bot for local development

### 2. Production Builds

**`make build-api-release`**
- Builds API for Linux ARM 64
- Strips debug symbols (`-w -s`) for smaller binaries
- Includes version info
- **Note**: Cannot be run on macOS due to CGO cross-compilation limitations (see CGO section below)

**`make build-bot-release`**
- Builds bot for Linux ARM 64
- Strips debug symbols (`-w -s`) for smaller binaries
- Includes version info
- **Note**: Cannot be run on macOS due to CGO cross-compilation limitations

**`make build-release`**
- Builds both binaries for production
- Creates VERSION file
- Used by GitHub Actions

### 3. Testing

**`make test`**
- Runs `go test -v ./...`

**`make vet`**
- Runs `go vet ./...`

**`make lint`**
- Runs `golangci-lint run ./...` (requires golangci-lint to be installed)

**`make ci-test`**
- Runs both `test` and `vet`
- Used by GitHub Actions

### 4. CI/CD

**`make ci-build`**
- Alias for `build-release`
- Used by GitHub Actions
- Builds both binaries for Linux

### 5. Utility

**`make deps`**
- Downloads Go module dependencies
- Automatically run by build targets

**`make clean`**
- Removes `./bin` directory

**`make info`**
- Shows current build configuration (VERSION, BUILD_TIME, GOOS, etc.)

**`make version-file`**
- Creates `bin/VERSION` with build metadata

**`make help`**
- Shows all available targets with descriptions

## Configuration Variables

All variables can be overridden via command line:

```bash
make build-api VERSION=v1.0.0 BUILD_TIME=20250109
```

### Available Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VERSION` | `dev` | Version string injected into binary |
| `BUILD_TIME` | Current timestamp | Build timestamp (YYYYMMDD-HHMMSS) |
| `GOOS` | Native OS | Target operating system |
| `GOARCH` | Native arch | Target architecture |
| `CGO_ENABLED` | `1` | CGO enabled (required for SQLite) |
| `BIN_DIR` | `./bin` | Output directory for binaries |

## CGO and Cross-Compilation

### Why CGO is Required

This project uses `github.com/mattn/go-sqlite3`, which requires CGO to interface with the native SQLite C library.

**Implications:**
- ✅ Local builds work natively (macOS → macOS, Linux → Linux)
- ❌ Cross-compilation with CGO is complex (macOS → Linux fails)
- ✅ GitHub Actions works (Linux → Linux, native compilation)

### Local Development (macOS/Windows)

```bash
# Works: native build
make build-api

# Fails: cross-compile with CGO from macOS to Linux
make build-api-release
# Error: undefined symbols for Linux syscalls
```

**Why it fails:**
- macOS doesn't have Linux system headers
- CGO requires native toolchain for target OS
- Cross-compilation with CGO needs additional setup

### CI/CD (GitHub Actions on Linux)

```bash
# Works: GitHub Actions runs on ubuntu-latest
make ci-build
```

**Why it works:**
- GitHub Actions uses Ubuntu (Linux)
- Native compilation (Linux → Linux)
- CGO works with native Linux toolchain

### Solution Options

If you need to build Linux binaries locally on macOS, you have three options:

#### Option 1: Use Docker
```bash
docker run --rm -v "$PWD":/app -w /app golang:1.24 make build-release
```

#### Option 2: Switch to Pure-Go SQLite
Replace `github.com/mattn/go-sqlite3` with `modernc.org/sqlite` (pure Go, no CGO):

```go
// Change this:
import _ "github.com/mattn/go-sqlite3"

// To this:
import _ "modernc.org/sqlite"
```

Then set `CGO_ENABLED=0` in Makefile:
```makefile
CGO_ENABLED := 0
```

**Trade-offs:**
- ✅ Works everywhere, no CGO needed
- ✅ Cross-compilation works
- ⚠️ Slightly slower than native SQLite (~10-20%)
- ⚠️ Requires code change and testing

#### Option 3: Accept Current Workflow
- Build locally for development (native builds work)
- Let GitHub Actions build production binaries (works)
- This is the current approach and works well

## GitHub Actions Integration

The `.github/workflows/release.yml` uses these Makefile targets:

```yaml
# Testing job
- name: Run tests
  run: make ci-test

# Build job
- name: Build binaries
  run: make ci-build VERSION=${{ steps.version.outputs.version }} BUILD_TIME=${{ steps.version.outputs.timestamp }}
```

**Benefits:**
- Single source of truth (Makefile)
- Easy to test locally: `make ci-test`, `make ci-build`
- Changes to build logic only need to update Makefile
- GitHub Actions workflow stays simple

## Common Workflows

### Local Development

```bash
# First time setup
make deps

# Build for local testing
make build-backend

# Run tests
make test

# Clean and rebuild
make clean && make build-backend
```

### Before Pushing to GitHub

```bash
# Run the same tests CI will run
make ci-test

# Check build configuration
make info
```

### CI/CD (Automated)

When you push to `main`:
1. GitHub Actions checks out code
2. Runs `make ci-test`
3. Generates version info
4. Runs `make ci-build VERSION=... BUILD_TIME=...`
5. Creates GitHub Release with binaries

### Manual Release Testing

```bash
# Simulate CI build (will fail on macOS, works on Linux)
make ci-build VERSION=test BUILD_TIME=20250109
```

## Troubleshooting

### Error: "undefined: main.Version"

**Cause:** Binary built without version info

**Fix:** Use Makefile targets (they include `-ldflags` automatically):
```bash
# Don't do this:
go build -o bin/api ./cmd/api

# Do this instead:
make build-api
```

### Error: "undefined reference to setresuid"

**Cause:** Trying to cross-compile with CGO from macOS to Linux

**Fix:** This is expected. Use one of these options:
1. Let GitHub Actions build Linux binaries
2. Use Docker to build (see Option 1 above)
3. Switch to pure-Go SQLite (see Option 2 above)

### Error: "golangci-lint: command not found"

**Cause:** `golangci-lint` not installed

**Fix:**
```bash
# Install golangci-lint (optional for local development)
brew install golangci-lint

# Or skip linting locally (CI will catch issues)
make test vet
```

### Build is slow

**Cause:** CGO compilation is slower than pure Go

**Context:** This is normal. CGO needs to compile C code and link libraries.

**Timing:**
- Local build: ~5-10 seconds
- CI build: ~10-15 seconds (includes tests)

## Best Practices

### 1. Always use Makefile targets

❌ Don't:
```bash
go build -o bin/api ./cmd/api
```

✅ Do:
```bash
make build-api
```

### 2. Override variables for custom builds

```bash
make build-api VERSION=$(git describe --tags) BUILD_TIME=$(date -u +%Y%m%d-%H%M%S)
```

### 3. Check configuration before building

```bash
make info
```

### 4. Clean between major changes

```bash
make clean && make build-backend
```

### 5. Test locally before pushing

```bash
make ci-test
```

## Makefile Structure

The Makefile is organized into sections:

1. **Configuration**: Variables and defaults
2. **Help**: `make help` target
3. **Dependencies**: `make deps`
4. **Testing & Linting**: `make test`, `make vet`, `make lint`
5. **Local Development Builds**: `make build-*-local`
6. **Release Builds**: `make build-*-release`
7. **Version File**: `make version-file`
8. **Web Frontend**: `make build-web`
9. **Combined Builds**: `make build`
10. **Clean**: `make clean`
11. **Development**: `make run-web`
12. **CI/CD**: `make ci-test`, `make ci-build`
13. **Info**: `make info`

Each section is clearly marked with comments for easy navigation.

## Future Improvements

Possible enhancements to consider:

1. **Add build caching**
   ```makefile
   GO_BUILD_CACHE ?= $(HOME)/.cache/go-build
   ```

2. **Add parallel builds**
   ```makefile
   build-release: build-api-release | build-bot-release
   ```

3. **Add install target**
   ```makefile
   install: build-backend
       cp $(API_BIN) $(GOPATH)/bin/
   ```

4. **Add coverage reporting**
   ```makefile
   coverage:
       go test -coverprofile=coverage.out ./...
       go tool cover -html=coverage.out
   ```

## Summary

The Makefile provides:
- ✅ Single source of truth for build commands
- ✅ Consistent builds (local & CI)
- ✅ Version injection built-in
- ✅ Clear separation (local vs release builds)
- ✅ Self-documenting (`make help`)
- ✅ Works with CGO requirements

For most development, you only need:
```bash
make build-backend  # Build locally
make test          # Test
make clean         # Clean
```

Let GitHub Actions handle production builds via `make ci-build`.
