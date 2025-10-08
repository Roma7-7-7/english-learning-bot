---
description: Analyze staged changes and create a commit with an appropriate message
---

You are creating a git commit based on the current conversation context and staged changes.

1. **Analyze Staged Changes:**
   - Run `git status` to see what files are staged
   - Run `git diff --cached` to see the actual changes
   - Understand the scope: is this a single focused change or multiple related changes?

2. **Review Conversation Context:**
   - Look back at the recent conversation to understand:
     - What was the user's goal or request?
     - What changes were made and why?
     - What problem was being solved?
   - This context is crucial for writing a meaningful commit message

3. **Determine Commit Type:**
   Based on the changes, identify the appropriate conventional commit type:
   - `feat:` - New feature or enhancement
   - `fix:` - Bug fix
   - `docs:` - Documentation only changes
   - `refactor:` - Code refactoring without behavior change
   - `test:` - Adding or updating tests
   - `chore:` - Maintenance tasks (deps, build, etc.)
   - `style:` - Code style/formatting changes
   - `perf:` - Performance improvements

4. **Craft Commit Message:**
   - **Format:** `type: brief description`
   - **Style:** Concise and clear (since you use squash PRs)
   - **Length:** Keep subject line under 72 characters
   - **Focus:** What changed, not why (the conversation provides context)
   - **Examples:**
     - `docs: restructure README and extract status tracking`
     - `feat: add queue-based auto-scaling controller`
     - `fix: correct Dockerfile references in Makefile`
     - `refactor: simplify error handling in API handlers`
     - `chore: update dependencies to latest versions`

5. **Create Commit:**
   - Stage any additional files if needed
   - Create commit using the format from CLAUDE.md:
     ```bash
     git commit -m "$(cat <<'EOF'
     type: brief description

     🤖 Generated with [Claude Code](https://claude.com/claude-code)

     Co-Authored-By: Claude <noreply@anthropic.com>
     EOF
     )"
     ```
   - Confirm the commit was created successfully

6. **Final Output:**
   - Show the commit message that was created
   - Confirm the commit hash
   - Remind user they can push with `git push` when ready

**Important Notes:**
- Keep it concise - detailed context is in the PR description
- Follow conventional commits format strictly
- Don't add multi-line descriptions unless absolutely necessary
- The commit message should make sense without reading the conversation
- If changes span multiple concerns, suggest splitting into separate commits
