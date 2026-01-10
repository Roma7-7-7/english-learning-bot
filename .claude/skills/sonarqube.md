---
description: SonarCloud/SonarQube code quality analysis and compliance
---

# SonarCloud Code Quality Analysis

This project uses **SonarCloud** for continuous code quality analysis across Go, TypeScript, and Shell scripts.

> **📝 Maintenance Note**: When encountering new SonarQube rules or patterns, add them to this skill file. See CLAUDE.md "Documentation Maintenance" section for guidelines.

## Checking Issues

### Public API (Recommended)

```bash
# Fetch all open/confirmed issues
curl -s "https://sonarcloud.io/api/issues/search?componentKeys=Roma7-7-7_english-learning-bot&statuses=OPEN,CONFIRMED&ps=100" | jq

# Fetch only new issues since last analysis
curl -s "https://sonarcloud.io/api/issues/search?componentKeys=Roma7-7-7_english-learning-bot&statuses=OPEN,CONFIRMED&sinceLeakPeriod=true&ps=100" | jq

# Format issues for readability (recommended)
curl -s "https://sonarcloud.io/api/issues/search?componentKeys=Roma7-7-7_english-learning-bot&statuses=OPEN,CONFIRMED&sinceLeakPeriod=true&ps=100" | \
  jq -r '.issues[] | "\(.rule) | \(.component) | Line \(.line // "N/A") | \(.message)"'

# Filter by specific language
curl -s "https://sonarcloud.io/api/issues/search?componentKeys=Roma7-7-7_english-learning-bot&statuses=OPEN,CONFIRMED&languages=go&ps=100" | \
  jq -r '.issues[] | "\(.rule) | \(.component) | Line \(.line // "N/A") | \(.message)"'

# Filter by specific rule
curl -s "https://sonarcloud.io/api/issues/search?componentKeys=Roma7-7-7_english-learning-bot&statuses=OPEN,CONFIRMED&rules=go:S3776&ps=100" | \
  jq -r '.issues[] | "\(.rule) | \(.component) | Line \(.line // "N/A") | \(.message)"'
```

### Web UI
- Visit: https://sonarcloud.io/project/issues?id=Roma7-7-7_english-learning-bot
- Requires authentication
- Filter by status: `OPEN`, `CONFIRMED`
- Filter by language: Go, TypeScript, Shell

## Common SonarQube Rules by Language

### Shell Scripts (`shelldre:*`)

**S7682 - Add explicit return statement**
- All functions must end with explicit `return 0` or `return 1`
- Applies even to simple print/echo functions
- See: `.claude/skills/shell-scripts.md` for patterns

**S7679 - Assign positional parameters**
- Positional parameters (`$1`, `$2`, etc.) must be assigned to local variables
- Example: `local message; message="$1"`
- See: `.claude/skills/shell-scripts.md` for patterns

### Go (`go:*`)

**S3776 - Cognitive Complexity**
- Default threshold: 15
- Refactor complex functions by:
  - Extracting helper functions
  - Simplifying nested conditions
  - Using early returns
  - Breaking down complex logic

**S1192 - String Literal Duplication**
- Don't repeat string literals more than 3 times
- Extract to constants or configuration
- Example: `const errorPrefix = "failed to process"`

**S1871 - Identical Branches**
- Two branches should not have the same implementation
- Merge identical cases or extract common logic

### TypeScript/React (`typescript:*`)

**S6819 - Accessibility**
- Use semantic HTML elements instead of ARIA roles
- Replace `<div role="button">` with `<button>`
- Use `<input type="button">`, `<input type="submit">`, etc.

**S3776 - Cognitive Complexity**
- Same as Go - refactor complex functions
- Common in React components with complex state logic

**S1128 - Unused Imports**
- Remove unused import statements
- Keep imports clean and minimal

## Fixing Issues Workflow

### 1. Check Current Issues
```bash
# See what's currently failing
curl -s "https://sonarcloud.io/api/issues/search?componentKeys=Roma7-7-7_english-learning-bot&statuses=OPEN,CONFIRMED&sinceLeakPeriod=true&ps=100" | \
  jq -r '.issues[] | "\(.rule) | \(.component) | Line \(.line // "N/A") | \(.message)"'
```

### 2. Identify the Rule
- Note the rule ID (e.g., `go:S3776`, `shelldre:S7682`)
- Look up the rule description in this file or online
- Understand what needs to be fixed

### 3. Apply the Fix
- Use language-specific patterns (see language skills)
- Test locally if possible
- Ensure fix doesn't break functionality

### 4. Verify Fix
- Commit and push changes
- Wait for CI/CD to run SonarCloud analysis
- Re-check issues using API

## CI/CD Integration

- **Automatic Analysis**: SonarCloud runs automatically on every push via GitHub Actions
- **No Manual Scans**: Local sonar-scanner is not configured (requires authentication)
- **PR Quality Gates**: New code must meet quality standards
- **Leak Period**: Focus on new issues in changed code

## Quality Gates

The project enforces these standards:
- **Coverage on New Code**: Not currently enforced
- **Duplications on New Code**: < 3%
- **Maintainability Rating**: A
- **Reliability Rating**: A
- **Security Rating**: A

## Best Practices

1. **Check Before PR**: Run the API command to see current issues before creating PR
2. **Fix New Issues**: Always fix issues you introduce (leak period issues)
3. **Language-Specific**: Follow the patterns in language-specific skills
4. **Don't Ignore**: Don't mark issues as "Won't Fix" without good reason
5. **Test Changes**: Ensure fixes don't break functionality

## Rule Reference Links

- Shell: https://rules.sonarsource.com/shell/
- Go: https://rules.sonarsource.com/go/
- TypeScript: https://rules.sonarsource.com/typescript/

## Common False Positives

### When to Mark as "Won't Fix"
- Generated code that can't be refactored
- Third-party code snippets
- Necessary complexity that can't be simplified
- Trade-offs where code clarity matters more than rule compliance

### How to Suppress
1. Add comment in code (language-specific)
2. Mark as "Won't Fix" in SonarCloud UI with explanation
3. Configure project-level rule exceptions in SonarCloud settings

## Troubleshooting

**Issue: "Component not found"**
- Check the component key is correct: `Roma7-7-7_english-learning-bot`
- Ensure the project is public or you're authenticated

**Issue: "Empty results"**
- Good news! No open issues
- Verify with different filters (remove `sinceLeakPeriod=true`)

**Issue: "Rule not found"**
- Check the rule ID syntax (e.g., `go:S3776` not `S3776`)
- Ensure the language plugin is enabled

## Project-Specific Notes

- **Main Branch**: `main`
- **SonarCloud Project**: https://sonarcloud.io/project/overview?id=Roma7-7-7_english-learning-bot
- **Languages Analyzed**: Go, TypeScript, JavaScript, Shell, CSS, HTML
- **Analysis Frequency**: On every push to any branch
