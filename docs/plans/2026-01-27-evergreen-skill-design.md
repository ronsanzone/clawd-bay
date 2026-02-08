# Evergreen CI Skill Design

**Date:** 2026-01-27
**Status:** Draft
**Author:** ron.sanzone + Claude

## Overview

A Claude Code skill for interacting with MongoDB's Evergreen CI system. Enables fetching patch/PR status, extracting test failures with verbatim errors, and mapping failures to local code for guided debugging.

## Goals

1. Fetch patch status from multiple entry points (Patch ID, PR URL)
2. Extract test failures with enough context to debug locally
3. Map failures to local codebase for quick navigation
4. Composable design for use by other skills/workflows

## Non-Goals

- Interactive/conversational workflow (batch output preferred)
- Task URL entry point (requires REST API complexity)
- Project-wide patch scanning

## Entry Points

### Supported Identifiers

| Input Type | Example | Detection |
|------------|---------|-----------|
| Patch ID | `69702928fd85d90007456df4` | 24-char hex regex |
| PR URL | `https://github.com/10gen/repo/pull/10595` | GitHub URL pattern |

### Identifier Normalization

```
Patch ID  → Use directly with evergreen CLI
PR URL    → Extract PR number, search list-patches for matching github_patch_data.pr_number
```

## Commands

### `/evergreen patch <patch_id|pr_url>`

Entry point - fetches patch overview and task statuses.

**Output:**
```
Patch: 69702928fd85d90007456df4
Status: failed
PR: 10gen/mms-automation#10595 (fix-toctou-race-condition)
Author: ron.sanzone
Duration: 18m 19s

Tasks:
  ✓ CheckGoFmt (code-health-linux)
  ✓ GolangciLint (code-health-linux)
  ✗ IntegrationTests (rhel80) - 3 failures
  ✗ UnitTests (ubuntu2204) - 1 failure
  ○ BuildAgent (amzn2) - pending

Failed Task IDs:
  - IntegrationTests: 6970abc123def456...
  - UnitTests: 6970xyz789ghi012...
```

**Implementation:**
```bash
# Normalize identifier to patch_id
if [[ "$ID" =~ ^[a-f0-9]{24}$ ]]; then
    PATCH_ID="$ID"
elif [[ "$ID" =~ github.com/.*/pull/([0-9]+) ]]; then
    PR_NUM="${BASH_REMATCH[1]}"
    PATCH_ID=$(evergreen list-patches -n 20 --json | jq -r \
        ".[] | select(.github_patch_data.pr_number == ${PR_NUM}) | .patch_id" | head -1)
fi

# Fetch patch details
evergreen list-patches -i "$PATCH_ID" --json
```

---

### `/evergreen failures <patch_id>`

Fetches all failures from a patch with verbatim error output.

**Output:**
```
Task: IntegrationTests (6970abc123def456...)
Variant: rhel80
Status: failed

Failures:
─────────────────────────────────────────
1. TestAuthTokenExpiration
   Duration: 2.3s

   Command:
     go test -v -run TestAuthTokenExpiration ./auth/...

   Error Output (verbatim):
     === RUN   TestAuthTokenExpiration
     --- FAIL: TestAuthTokenExpiration (2.30s)
         token_test.go:142: assertion failed: expected token.Valid() to be true
             got: false
             token.ExpiresAt: 2026-01-20T10:00:00Z
             time.Now():      2026-01-20T10:00:01Z
         token_test.go:143: token unexpectedly expired during grace period
     FAIL

   Stack:
     auth/token_test.go:142
     auth/token_test.go:89

2. TestConnectionPoolDrain
   Duration: 0.8s

   Command:
     go test -v -timeout 30s -run TestConnectionPoolDrain ./pool/...

   Error Output (verbatim):
     === RUN   TestConnectionPoolDrain
     --- FAIL: TestConnectionPoolDrain (0.80s)
         manager_test.go:234: context deadline exceeded
             waiting for 5 connections to close
             open connections: 3
             state: draining
     FAIL

   Stack:
     pool/manager_test.go:234
```

**Implementation:**
```bash
# For each failed task in patch:
evergreen task build TaskLogs --task_id "$TASK_ID" --type task_log

# Get test-specific logs
evergreen task build TestLogs --task_id "$TASK_ID" --log_path "$TEST_LOG_PATH"
```

---

### `/evergreen debug <task_id>`

Maps a single task's failures to local code paths.

**Output:**
```
Failure: TestAuthTokenExpiration

  Error (verbatim first 5 lines):
    token_test.go:142: assertion failed: expected token.Valid() to be true
        got: false
        token.ExpiresAt: 2026-01-20T10:00:00Z
        time.Now():      2026-01-20T10:00:01Z
    token_test.go:143: token unexpectedly expired during grace period

  Local files:
    → Test:   /Users/ron/code/mms-automation/auth/token_test.go:142
    → Source: /Users/ron/code/mms-automation/auth/token.go:78 (Valid method)

  Investigate:
    - Grace period logic in token.Valid()
    - Time comparison at token.go:82-85
```

**Implementation:**
```bash
# Fetch test logs
LOGS=$(evergreen task build TestLogs --task_id "$TASK_ID")

# Parse for file:line references
# Map to local paths using current directory as repo root
# Use grep/rg to locate files and extract context
```

## Output Contract for Automation

To enable reliable parsing by other skills:

| Element | Format |
|---------|--------|
| Failed task IDs | Under `Failed Task IDs:` header |
| Verbatim errors | In `Error Output (verbatim):` block |
| Local file paths | Prefixed with `→ Test:` or `→ Source:` |
| Commands | In `Command:` block |

## Composability

### Calling from Other Skills

```markdown
1. Run `/evergreen patch <id>` to get failed task IDs
2. Run `/evergreen failures <patch_id>` to get error details
3. Use error output to guide code investigation
4. Apply fixes
5. Re-run tests locally before pushing
```

### Example Workflow: CI Fix Pipeline

```
┌─────────────────────────────────────────────────────────┐
│  /fix-ci <pr_url>                                       │
│                                                         │
│  1. /evergreen patch <pr_url>                           │
│       → Get failed task IDs                             │
│                                                         │
│  2. /evergreen failures <patch_id>                      │
│       → Get verbatim errors + commands                  │
│                                                         │
│  3. /evergreen debug <task_id>  (for each failure)      │
│       → Map to local files                              │
│                                                         │
│  4. Read local files, analyze failures                  │
│       → Propose fixes                                   │
│                                                         │
│  5. Run test command locally                            │
│       → Verify fix before pushing                       │
└─────────────────────────────────────────────────────────┘
```

## File Structure

```
.claude/skills/evergreen/SKILL.md
```

## Dependencies

- `evergreen` CLI installed and configured (`~/.evergreen.yml`)
- `jq` for JSON parsing
- Current directory is the relevant repo

## Error Handling

| Condition | Response |
|-----------|----------|
| `evergreen` not in PATH | "Install: evergreen get-update" |
| `~/.evergreen.yml` missing | "Run: evergreen client setup" |
| Patch not found | "Patch ID not found. Check ID or use list-patches" |
| PR has no Evergreen patch | "No Evergreen patch found for PR #X" |
| All tasks passed | "No failures - all X tasks passed ✓" |

## Open Questions

1. **Log size management** - Large test logs may exceed context limits. Consider:
   - Tail last N lines of error output
   - Summarize if over threshold
   - Offer "full logs" sub-command

2. **Test log path discovery** - How to reliably find test log paths within a task's logs directory? May need exploration of actual Evergreen task output structure.

3. **Multi-module patches** - Patches with module code changes may have failures in different repos. Current design assumes single-repo scope.

## Next Steps

1. Implement skill file at `.claude/skills/evergreen/SKILL.md`
2. Test with real patch IDs from recent PRs
3. Iterate on log parsing based on actual Evergreen output format
4. Build example `/fix-ci` workflow skill on top
