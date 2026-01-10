---
description: Shell script development guidelines and best practices
---

# Shell Script Development Guidelines

This project follows strict shell scripting standards for maintainability, safety, and code quality.

## Required Function Pattern

All shell functions in this project must follow this pattern:

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

### Key Requirements

1. **Declare locals first**: `local variable_name`
2. **Assign parameters separately**: `variable_name="$1"`
3. **Explicit return**: Always end with `return 0` or appropriate exit code
4. **Named variables**: Never use `$1`, `$2` directly in logic

### Why This Pattern?

**Q: Why declare and assign separately instead of `local var="$1"`?**

A: Two reasons:
1. **SonarQube Compliance**: Rule `shelldre:S7679` requires positional parameters to be explicitly assigned
2. **Error Handling**: If assignment fails, you can detect it separately from declaration

**Q: Why explicit return statements?**

A: Three reasons:
1. **SonarQube Compliance**: Rule `shelldre:S7682` requires all functions to have explicit returns
2. **Clarity**: Makes exit status intentional and clear
3. **Debugging**: Easier to trace function exit points

**Q: Why local variables in each function instead of globals?**

A: Multiple reasons:
1. **Scope Isolation**: Each function's variables are independent - no conflicts
2. **Safety**: Prevents accidental global variable pollution
3. **Thread Safety**: If scripts run in parallel, locals prevent race conditions
4. **Maintainability**: Self-contained functions are easier to understand and test
5. **Best Practice**: This is the industry-standard pattern

**Example showing locals don't conflict:**
```bash
func1() {
    local message
    message="$1"
    echo "func1: $message"
    return 0
}

func2() {
    local message  # Different scope - no conflict with func1
    message="$1"
    echo "func2: $message"
    return 0
}

func1 "hello"  # Outputs: func1: hello
func2 "world"  # Outputs: func2: world
```

## Complete Examples

### Simple Function (One Parameter)

```bash
log() {
    local message
    message="$1"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $message"
    return 0
}
```

### Multi-Parameter Function

```bash
log_colored() {
    local message
    local color
    message="$1"
    color="$2"
    echo -e "${color}[$(date '+%Y-%m-%d %H:%M:%S')] $message${NC}"
    return 0
}
```

### Function with Logic and Error Handling

```bash
download_file() {
    local filename
    local url
    filename="$1"
    url="$2"
    
    if [[ -z "$filename" ]]; then
        echo "Error: filename is required" >&2
        return 1
    fi
    
    if ! curl -sfL "$url" -o "$filename"; then
        echo "Error: failed to download $url" >&2
        return 1
    fi
    
    return 0
}
```

### Function with Optional Parameters

```bash
process_data() {
    local input_file
    local output_file
    local verbose
    input_file="$1"
    output_file="${2:-output.txt}"  # Default value if $2 is empty
    verbose="${3:-false}"
    
    if [[ "$verbose" == "true" ]]; then
        echo "Processing $input_file -> $output_file"
    fi
    
    # Processing logic here
    
    return 0
}
```

## General Best Practices

### 1. Script Header

```bash
#!/bin/bash
set -e  # Exit on error
set -u  # Exit on undefined variable (optional, use carefully)
set -o pipefail  # Pipe failures cause exit
```

### 2. Variable Quoting

```bash
# ✅ GOOD - Always quote variables
echo "$variable"
cp "$source" "$destination"

# ❌ BAD - Unquoted (word splitting issues)
echo $variable
cp $source $destination
```

### 3. Test Constructs

```bash
# ✅ GOOD - Use [[ ]] for tests
if [[ -f "$file" ]]; then
    echo "File exists"
fi

# ❌ BAD - Old [ ] syntax (less safe)
if [ -f "$file" ]; then
    echo "File exists"
fi
```

### 4. Error Handling

```bash
# Check command success inline
if ! command_that_might_fail; then
    echo "Error: command failed" >&2
    return 1
fi

# Use trap for cleanup
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT
```

### 5. String Comparison

```bash
# ✅ GOOD - Proper string comparison
if [[ "$status" == "active" ]]; then
    echo "Active"
fi

# ❌ BAD - Using = instead of ==
if [[ "$status" = "active" ]]; then
    echo "Active"
fi
```

### 6. Arrays

```bash
# Declare array
declare -a files
files=("file1.txt" "file2.txt" "file3.txt")

# Iterate array
for file in "${files[@]}"; do
    echo "Processing $file"
done
```

## Project Shell Scripts

- `deployment/deploy.sh` - Automated deployment (downloads and installs binaries)
- `deployment/backup.sh` - Database backup to S3 (EC2 deployments)
- `deployment/setup-ec2.sh` - EC2 initial setup
- `deployment/setup-simple.sh` - Simple deployment setup
- `run-api.sh` - Local API runner (development)
- `run-bot.sh` - Local bot runner (development)

## Testing Shell Scripts

### Syntax Check
```bash
bash -n script.sh
```

### Run with Tracing
```bash
bash -x script.sh
```

### ShellCheck (if installed)
```bash
shellcheck script.sh
```

### Manual Testing
- Test with valid inputs
- Test with missing/invalid inputs
- Test error conditions
- Verify cleanup happens on exit

## Common Patterns in This Project

### Logging Functions
All deployment scripts use consistent logging:
```bash
log() {
    local message
    message="$1"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $message" | tee -a "$LOG_FILE"
    return 0
}
```

### Color Output
```bash
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Success${NC}"
echo -e "${RED}Error${NC}"
```

### Architecture Detection
```bash
ARCH=$(uname -m)
case "${ARCH}" in
    x86_64)
        ARCH_SUFFIX="amd64"
        ;;
    aarch64|arm64)
        ARCH_SUFFIX="arm64"
        ;;
    *)
        echo "Unsupported architecture: ${ARCH}" >&2
        exit 1
        ;;
esac
```

## When Writing New Shell Scripts

1. ✅ Start with `set -e` for safety
2. ✅ Use functions for reusable logic
3. ✅ Follow the required function pattern (locals, explicit returns)
4. ✅ Quote all variables
5. ✅ Use `[[ ]]` for tests
6. ✅ Add error handling for critical operations
7. ✅ Use `trap` for cleanup
8. ✅ Test thoroughly before committing

## Code Quality

Shell scripts in this project are analyzed by **SonarCloud**. See `.claude/skills/sonarqube.md` for:
- How to check for issues
- Common SonarQube rules for shell scripts
- How to fix compliance issues

## Resources

- [Bash Reference Manual](https://www.gnu.org/software/bash/manual/)
- [ShellCheck](https://www.shellcheck.net/)
- [Google Shell Style Guide](https://google.github.io/styleguide/shellguide.html)
