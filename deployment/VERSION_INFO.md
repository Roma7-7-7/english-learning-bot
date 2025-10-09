1# Version Information System

## Overview

The application now includes build-time version information that's injected during compilation and exposed through logs and HTTP endpoints.

## How It Works

### 1. Go Linker Flags (`-ldflags -X`)

The `-X` flag tells the Go linker to set the value of a string variable at **build time**:

```bash
go build -ldflags="-X main.Version=v1.2.3 -X main.BuildTime=2025-01-09"
```

This sets:
```go
var Version = "v1.2.3"      // set at build time
var BuildTime = "2025-01-09" // set at build time
```

**Key benefits:**
- No code changes needed between releases
- Version info is baked into the binary
- Can be set differently for each build (dev, staging, prod)
- Zero runtime overhead

### 2. Variable Declaration

In both `cmd/api/main.go` and `cmd/bot/main.go`:

```go
var (
    // Version is set via -ldflags at build time
    Version = "dev"
    // BuildTime is set via -ldflags at build time
    BuildTime = "unknown"
)
```

**Default values** (`"dev"` and `"unknown"`):
- Used when building locally without `-ldflags`
- Makes it obvious you're running a dev build
- Prevents empty strings which could be confusing

### 3. Startup Logs

Both services now log version info on startup:

**API Server:**
```json
{
  "time": "2025-01-09T10:30:00Z",
  "level": "INFO",
  "msg": "starting api server",
  "version": "v20250109-103000-abc1234",
  "build_time": "20250109-103000",
  "address": ":8080"
}
```

**Telegram Bot:**
```json
{
  "time": "2025-01-09T10:30:00Z",
  "level": "INFO",
  "msg": "starting bot",
  "version": "v20250109-103000-abc1234",
  "build_time": "20250109-103000",
  "config": {...}
}
```

**Why this is useful:**
- Immediately know which version is running
- Correlate logs with deployments
- Debug "which version has this bug?"
- Audit trail for compliance

### 4. Health Endpoint

The API now exposes `GET /health`:

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "ok",
  "version": "v20250109-103000-abc1234",
  "build_time": "20250109-103000"
}
```

**Use cases:**
- Load balancer health checks
- Monitoring systems (Prometheus, DataDog, etc.)
- Quick verification: "Is my deployment successful?"
- CI/CD validation: "Did the new version deploy?"

## Version Format

The GitHub Actions workflow creates versions in this format:

```
v20250109-103000-abc1234
 ├─────────┘ └────┘ └─────┘
 │          │       └─ git commit SHA (short)
 │          └───────── time (HHMMSS)
 └──────────────────── date (YYYYMMDD)
```

**Example:** `v20250109-103000-a1b2c3d`
- Built on: January 9, 2025 at 10:30:00 UTC
- From commit: `a1b2c3d`

**Why this format:**
- **Chronologically sortable**: Newer versions > older versions
- **Unique**: Timestamp + SHA prevents collisions
- **Traceable**: SHA links back to exact commit
- **Human-readable**: Can quickly tell when it was built

## Testing Locally

### Build with version info:
```bash
# Manual build with custom version
go build -ldflags="-X main.Version=local-test -X main.BuildTime=$(date -u +%Y%m%d-%H%M%S)" -o bin/english-learning-api ./cmd/api

# Run and check logs
./bin/english-learning-api
# Should see: version=local-test build_time=20250109-103000

# Check health endpoint
curl http://localhost:8080/health
```

### Build without version info (uses defaults):
```bash
go build -o bin/english-learning-api ./cmd/api
./bin/english-learning-api
# Should see: version=dev build_time=unknown
```

## Production Usage

### Via GitHub Actions (automatic)

When you push to `main`, GitHub Actions automatically:

1. **Generates version info:**
   ```yaml
   version=$(git rev-parse --short HEAD)           # abc1234
   timestamp=$(date -u +%Y%m%d-%H%M%S)            # 20250109-103000
   tag=v${timestamp}-${version}                    # v20250109-103000-abc1234
   ```

2. **Builds with version baked in:**
   ```bash
   go build -ldflags="-X main.Version=abc1234 -X main.BuildTime=20250109-103000" ...
   ```

3. **Creates GitHub Release** with that tag

4. **EC2 downloads and runs** the versioned binary

### Checking deployed version

**On EC2:**
```bash
# Check current deployed version
cat /opt/english-learning-bot/current_version

# Check via logs
sudo journalctl -u english-learning-api.service | grep "starting api server"

# Check via health endpoint
curl http://localhost:8080/health
```

**From your machine:**
```bash
# If API is publicly accessible
curl http://your-ec2-ip:8080/health
```

## Troubleshooting

### Version shows as "dev" in production

**Cause:** Binary was built without `-ldflags`

**Fix:** Ensure GitHub Actions workflow is running properly:
```bash
# Check latest release
curl -s https://api.github.com/repos/Roma7-7-7/english-learning-bot/releases/latest | grep tag_name

# Check GitHub Actions runs
# Visit: https://github.com/Roma7-7-7/english-learning-bot/actions
```

### Version doesn't match what I pushed

**Cause:** Deployment hasn't run yet (runs hourly)

**Fix:** Manually trigger deployment:
```bash
sudo /opt/english-learning-bot/deploy.sh
```

### Want to see full version info

```bash
# API startup log (full details)
sudo journalctl -u english-learning-api.service -n 50 | grep "starting api server"

# Bot startup log
sudo journalctl -u english-learning-bot.service -n 50 | grep "starting bot"

# Version file (just the tag)
cat /opt/english-learning-bot/current_version

# Detailed version file (created during build)
cat /opt/english-learning-bot/bin/VERSION
```

## Advanced: Custom Version Schemes

If you want semantic versioning (v1.2.3) instead:

### Option 1: Git Tags
```yaml
# In .github/workflows/release.yml, change:
tag=v$(date -u +%Y%m%d-%H%M%S)-$(git rev-parse --short HEAD)

# To:
tag=$(git describe --tags --always)
```

### Option 2: Manual Versions
```yaml
# Use a VERSION file in repo
tag=v$(cat VERSION)
```

### Option 3: Hybrid
```yaml
# Semantic version + commit SHA
tag=v1.2.3-$(git rev-parse --short HEAD)
```

## Monitoring Integration Examples

### Prometheus

If you add Prometheus metrics later, you can expose version as a gauge:

```go
// In your metrics setup
versionGauge := prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "app_version_info",
        Help: "Application version information",
    },
    []string{"version", "build_time"},
)
versionGauge.WithLabelValues(Version, BuildTime).Set(1)
```

### Datadog

Health endpoint can be polled by Datadog:

```yaml
# datadog-agent config
instances:
  - url: http://localhost:8080/health
    tags:
      - service:english-learning-api
```

### CloudWatch

Parse version from logs:

```json
{
  "filterPattern": "[timestamp, request_id, level, msg, version=version*, build_time=build_time*, ...]",
  "metricTransformations": [{
    "metricName": "AppVersion",
    "metricNamespace": "EnglishBot",
    "metricValue": "1"
  }]
}
```

## Summary

✅ **Startup logs** include version info
✅ **Health endpoint** returns version info
✅ **Automatic versioning** via GitHub Actions
✅ **Traceable** back to exact commit
✅ **Zero runtime overhead** (set at compile time)
✅ **Works locally** with defaults (`dev` / `unknown`)

This setup is production-ready and follows industry best practices for version tracking and observability.
