---
description: Shell script development guidelines and SonarQube compliance
---

# Shell Script Development Guidelines

This project enforces strict code quality standards for shell scripts using SonarCloud analysis.

## SonarQube Compliance

### Required Patterns

All shell functions MUST follow these patterns to pass SonarQube analysis:

#### Rule: shelldre:S7679 - Assign Positional Parameters
Positional parameters (`$1`, `$2`, etc.) must be assigned to local variables before use.

```bash
# ❌ BAD - Direct positional parameter usage
bad_function() {
    echo "Message: $1"
}

# ✅ GOOD - Assign to local variable first
good_function() {
    local message
    message="$1"
    echo "Message: $message"
}
```

#### Rule: shelldre:S7682 - Explicit Return Statements
All functions must have an explicit `return` statement.

```bash
# ❌ BAD - No explicit return
bad_function() {
    local message
    message="$1"
    echo "Message: $message"
}

# ✅ GOOD - Explicit return statement
good_function() {
    local message
    message="$1"
    echo "Message: $message"
    return 0
}
```

### Complete Function Template

```bash
function_name() {
    local param1
    local param2
    param1="$1"
    param2="$2"
    
    # Function logic here
    echo "Param1: $param1, Param2: $param2"
    
    return 0
}
```

## Why Local Variables in Each Function?

**Q: Why not make variables global to avoid repetition?**

**A: Each function should declare its own local variables because:**

1. **Scope Isolation**: Local variables are scoped to their function - no conflicts between functions
2. **Safety**: Prevents accidental global variable pollution
3. **Maintainability**: Each function is self-contained and easier to understand
4. **Thread Safety**: If scripts ever run in parallel, locals prevent race conditions
5. **Best Practice**: This is the recommended pattern in shell scripting

**Example showing why locals don't conflict:**
```bash
func1() {
    local message
    message="$1"
    echo "func1: $message"
    return 0
}

func2() {
    local message  # This doesn't conflict with func1's message
    message="$1"
    echo "func2: $message"
    return 0
}
```

## Checking SonarCloud Issues

### Public API (Recommended for Automation)

```bash
# Fetch all open/confirmed issues
curl -s "https://sonarcloud.io/api/issues/search?componentKeys=Roma7-7-7_english-learning-bot&statuses=OPEN,CONFIRMED&ps=100" | jq

# Fetch only new issues since last analysis
curl -s "https://sonarcloud.io/api/issues/search?componentKeys=Roma7-7-7_english-learning-bot&statuses=OPEN,CONFIRMED&sinceLeakPeriod=true&ps=100" | jq

# Format issues for readability
curl -s "https://sonarcloud.io/api/issues/search?componentKeys=Roma7-7-7_english-learning-bot&statuses=OPEN,CONFIRMED&sinceLeakPeriod=true&ps=100" | \
  jq -r '.issues[] | "\(.rule) | \(.component) | Line \(.line // "N/A") | \(.message)"'
```

### Web UI
- Visit: https://sonarcloud.io/project/issues?id=Roma7-7-7_english-learning-bot
- Requires authentication

## Common Shell Script Rules

- `shelldre:S7682` - Functions must have explicit `return` statements
- `shelldre:S7679` - Positional parameters must be assigned to local variables
- `shellcheck:SC2086` - Double quote to prevent globbing and word splitting
- `shellcheck:SC2181` - Check exit code directly with `if mycmd;`, not `$?`

## Project Shell Scripts

- `deployment/deploy.sh` - Automated deployment script
- `deployment/backup.sh` - Database backup to S3
- `deployment/setup-ec2.sh` - EC2 initial setup
- `deployment/setup-simple.sh` - Simple deployment setup
- `run-api.sh` - Local API runner
- `run-bot.sh` - Local bot runner

## When Writing New Shell Scripts

1. Start with strict mode: `set -e` (exit on error)
2. Use functions for reusable logic
3. Always assign positional parameters to named local variables
4. Always add explicit `return` statements
5. Double-quote variables to prevent word splitting
6. Use `[[ ]]` instead of `[ ]` for tests
7. Prefer `local var; var="$1"` over `local var="$1"` for SonarQube compliance

## Testing Shell Scripts Locally

```bash
# Syntax check
bash -n script.sh

# Run with tracing
bash -x script.sh

# ShellCheck (if installed)
shellcheck script.sh
```

## Note on CI/CD

SonarCloud analysis runs automatically via GitHub Actions. Manual local scans are not required for development.
