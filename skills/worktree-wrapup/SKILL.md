---
name: worktree-wrapup
description: Wrap up a completed development worktree by checking repository state, summarizing and grouping changes into logical commits, committing with Codex co-author attribution, merging the worktree branch back into the main branch, and cleaning up the worktree safely. Use when the user asks to finalize, land, or close out work from a git worktree.
---

# Worktree Wrap-Up

Finalize work in this order:
1. Inspect working tree state and branch safety.
2. Summarize changed files and propose logical commit boundaries.
3. Create one or more commits with Codex co-author attribution.
4. Merge worktree branch into main branch.
5. Remove the worktree and optionally delete the feature branch.

Use `scripts/summarize_changes.sh` first to produce a quick change summary.

## Required Inputs

- `main_branch`: Default `main`.
- `coauthor_name`: Default `Codex`.
- `coauthor_email`: Ask user; if unavailable, use `codex@openai.com`.
- `cleanup_branch`: Whether to delete merged feature branch after cleanup.

## Workflow

### 1) Preflight checks

Run from inside the worktree root.

```bash
git rev-parse --show-toplevel
git branch --show-current
git status --short
```

Fail fast if:
- Current branch equals `main_branch`.
- Branch is detached.
- There are unresolved conflicts.

### 2) Summarize and determine logical commits

Run:

```bash
scripts/summarize_changes.sh
```

Then create a commit plan:
- Group by behavior or concern, not by file type.
- Keep refactors separate from feature or bugfix commits.
- Keep generated/format-only changes separate when possible.
- Ensure each commit is buildable/testable when feasible.

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

After committing, verify:

```bash
git status --short
git log --oneline --decorate -n 10
```

### 4) Merge into main

Update main and merge from the repository that owns both branches:

```bash
repo_root="$(git rev-parse --show-toplevel)"
feature_branch="$(git branch --show-current)"

git -C "$repo_root" checkout main
git -C "$repo_root" pull --ff-only
git -C "$repo_root" merge --no-ff "$feature_branch"
```

If conflicts occur, resolve them and continue:

```bash
git add <resolved files>
git commit
```

### 5) Cleanup worktree

Only after successful merge:

```bash
worktree_path="$(git rev-parse --show-toplevel)"
feature_branch="$(git -C "$worktree_path" branch --show-current)"

git worktree remove "$worktree_path"
git branch -d "$feature_branch"
```

Use `git branch -D` only when the user explicitly asks to force-delete.

## Output Format

Return:
1. Preflight status summary.
2. Proposed logical commit plan.
3. Exact commit commands executed (with co-author line).
4. Merge result and resulting commit SHA.
5. Cleanup result (worktree removed, branch deleted or retained).
