---
description: Prepare current PR for review by verifying completeness and updating documentation
---

You are preparing a GitHub Pull Request for review. Follow these steps:

1. **Identify PR Context:**
   - Use `gh pr view --json number,title,body,headRefName` to get current PR details
   - Review CLAUDE.md and README.md for relevant context

2. **Analyze Planned Work:**
   - Check CLAUDE.md and README.md for context about what was planned
   - Infer from PR title and commit messages
   - Use `gh pr view --json commits` to see commit history if needed

3. **Verify Implementation Completeness:**
   - Run `git diff main...HEAD` to see all changes in the PR
   - Compare changes against planned work to identify:
     - ✅ Completed items
     - ❌ Missing implementations
     - ⚠️  Partial implementations
   - Check if CLAUDE.md needs updates (new commands, dependencies, patterns)
   - Check if README.md needs updates (new features, API endpoints, deployment changes)

4. **Update Documentation:**
   - Update CLAUDE.md if new patterns, architecture changes, or dependencies were added
   - Update README.md if user-facing features, API endpoints, or deployment process changed
   - Update schema documentation if database changes were made
   - Ensure all code examples are accurate

5. **Quality Checks:**
   - Verify Go code: `make build` and `go test ./...` would pass
   - Check frontend: `cd web && npm run lint` would pass
   - Ensure proper error handling patterns (structured logging with slog)
   - Check if tests were added for new functionality
   - Check database schema changes are in SQLite files
   - Verify API endpoint changes match OpenAPI/documentation
   - Check Telegram bot command changes are properly registered

6. **Prepare PR Description:**
   - Create a CONCISE PR description that can be read in ~5 minutes maximum
   - **CRITICAL**: Only document changes visible in `git diff main...HEAD`
   - **Issue References:**
     - Check PR title and commit messages for issue references (e.g., "#73", "issue 73")
     - If issue numbers are mentioned, add proper GitHub linking keywords to PR description:
       - Use `Fixes #<number>`, `Closes #<number>`, or `Resolves #<number>` to auto-close issues
       - Or use `#<number>` to link without auto-closing
     - Add issue references in the description body, not the title
   - **DO NOT include**:
     - Intermediate fixes made during development that don't appear in final diff
     - Implementation details from conversation that were changed/reverted
     - Bug fixes that were part of the same PR and never existed in main
     - Verbose explanations or step-by-step testing procedures
   - **DO include**:
     - **Summary:** 2-3 sentence overview of what was added/changed
     - **Key Changes:** Bulleted list organized by component/area (be brief!)
     - **Testing:** Minimal commands to verify functionality
     - **Related:** Links to relevant documentation and issues
   - Use markdown formatting for clarity
   - Keep it scannable - reviewers should understand changes in 5 minutes

7. **Push and Update PR:**
   - If documentation was updated, stage and commit changes:
     - `git add <files>`
     - `git commit -m "docs: update documentation for PR"`
   - Push changes: `git push`
   - Update PR title and description: `gh pr edit <number> --title "..." --body "..."`

8. **Final Summary:**
   - Provide a summary of:
     - What was completed vs planned
     - What documentation was updated
     - Any gaps or follow-up work needed
     - Link to the PR for easy access

**Important Notes:**
- **Keep PR description CONCISE** - Target 5 minute read time
- **Only describe final state** - Don't document the development journey or intermediate fixes
- **Focus on what reviewers will see** - If it's not in `git diff main...HEAD`, don't mention it
- If something is missing from the implementation, clearly flag it
- Don't make assumptions - verify by reading actual code changes
- Ensure PR description is clear enough for reviewers who aren't familiar with the codebase
- Follow conventional commit style for any documentation commits
