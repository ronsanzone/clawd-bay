---
name: commit
description: Create high-quality git commits in any repository by checking for uncommitted changes, summarizing and grouping diffs into logical commit units, and committing with Codex co-author attribution. Use when the user asks to commit current changes, split changes into logical commits, or prepare clean commit history without running a full worktree merge workflow.
---

# Commit

Finalize local changes into logical commits in this order:
1. Inspect repository state and commit safety.
2. Summarize changed files and propose logical commit boundaries.
3. Stage and commit each logical group with Codex co-author attribution.
4. Verify a clean working tree or explain remaining files.

Use `scripts/summarize_changes.sh` first to produce a quick change summary.

## Required Inputs

- `coauthor_name`: Default `Codex`.
- `coauthor_email`: Ask user; if unavailable, use `codex@openai.com`.
- `commit_style`: Prefer Conventional Commits unless user requests another style.

## Workflow

### 1) Preflight checks

Run from the target repository root or subdirectory.

```bash
git rev-parse --show-toplevel
git branch --show-current
git status --short
```

Fail fast if:
- Branch is detached.
- There are unresolved conflicts.
- No user-visible changes exist to commit.

### 2) Summarize and determine logical commits

Run:

```bash
scripts/summarize_changes.sh
```

Then create a commit plan:
- Group by behavior or concern, not by file type.
- Keep refactors separate from feature or bugfix commits.
- Keep generated/format-only changes separate when possible.
- Keep each commit independently understandable.

Show the plan before committing:
- Commit title per group.
- Files in each group.
- Why each group is independent.

### 3) Commit with Codex attribution

For each logical group:

```bash
git add <files for group>
git commit -m "<type(scope): summary>" -m "Co-authored-by: Codex <codex@openai.com>"
```

If the user gave a different co-author, use:

```bash
git commit -m "<type(scope): summary>" -m "Co-authored-by: <name> <email>"
```

After each commit, verify:

```bash
git status --short
git log --oneline --decorate -n 10
```

### 4) Post-commit verification

Confirm whether changes remain:

```bash
git status --short
```

If files remain uncommitted, explain what is left and propose next commit groups.

## Output Format

Return:
1. Preflight status summary.
2. Proposed logical commit plan.
3. Exact commit commands executed (with co-author line).
4. Resulting commit SHAs.
5. Remaining uncommitted files, if any.
