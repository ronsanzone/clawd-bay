# ClawdBay — Configured Project Scope and Worktree Discovery Design

**Date:** 2026-02-14
**Status:** Draft

## Summary

This design replaces brittle prefix/path-guess discovery for dashboard grouping with persisted, explicit project configuration.

The system will:

1. Persist configured project roots in TOML.
2. Show only configured projects in the TUI and `cb list`.
3. Canonicalize all paths via symlink resolution (required).
4. Auto-detect worktrees using git worktree metadata (supports nested slash-branch paths under `.worktrees/`).
5. Preserve hierarchy in outputs: `Project -> Worktree -> Session -> Window`.

## Problem Statement

Current grouping and visibility rely on runtime inference from tmux pane paths and session naming patterns. This causes noise and brittleness:

1. Unwanted repos can appear in views.
2. Grouping quality depends on pane cwd and naming conventions.
3. Prefix/path matching is fragile when directory layouts vary.

## Goals

1. Add persistent settings for project visibility scope.
2. Make visibility deterministic and user-controlled.
3. Preserve `.worktrees` hierarchy in UI and listing output.
4. Keep live status/session behavior driven by tmux runtime data.
5. Apply the same scope model to both `cb dash` and `cb list`.

## Non-Goals

1. Auto-discover new root projects outside explicit configuration.
2. Add database-backed state.
3. Change session status semantics (`WORKING > WAITING > IDLE > DONE` remains unchanged).
4. Change attach semantics in dashboard selection flow.

## User Decisions Captured

1. "Inactive" means there are no active tmux agent sessions for that project/worktree.
2. Do not show anything outside configured projects.
3. Symlink-aware canonical paths are required.
4. `cb list` should use the same configured project filtering.
5. Configuration format is TOML.
6. Worktree hierarchy should preserve `.worktrees` structure, and sessions running from the main repo directory must remain visible.
7. `cb project remove` should require canonical path by default; name-based removal must be explicit.
8. Empty project configuration is valid.
9. `cb start` should warn when the current repo is not configured.
10. `cb clist` remains unchanged and is not project-scoped in this feature.
11. No legacy fallback discovery mode is needed.

## UX Model

### High-level behavior

1. A project appears only if its canonical path is configured.
2. Under each project, worktrees are discovered via `git -C <project> worktree list --porcelain`, filtered to include:
   1. the canonical project root (rendered as a synthetic "main repo" worktree node), and
   2. canonical paths under `<project>/.worktrees/`.
3. Sessions are associated to discovered worktrees by canonical path containment/equality (longest match wins).
4. If a session is inside the configured project root but not under a discovered `.worktrees` path, it is attached to the synthetic main-repo worktree node.
5. Windows and statuses remain derived from tmux runtime inspection.

### TUI hierarchy

1. Project node
2. Worktree node
3. Session node
4. Window node

If a worktree exists but has no active session, it remains visible as an inactive leaf in the project hierarchy.  
The synthetic main-repo worktree node is always available for sessions running from the repository root (outside `.worktrees`).

### `cb list` hierarchy

Text output becomes project-scoped with worktree grouping. Example:

```text
repo-a
  (main repo)
    cb_repo-maintenance             1 window   (IDLE)
  .worktrees/repo-a-feat-auth
    cb_feat-auth                    2 windows  (WORKING)
  .worktrees/repo-a-refactor
    (no active session)
```

## Configuration Design

### File location

`~/.config/cb/config.toml`

### Schema (v1)

```toml
version = 1

[[projects]]
path = "/Users/sanzoner/code/repo-a"
name = "repo-a" # optional display name override
```

### Validation rules

1. `version` must be supported (`1` initially).
2. `projects` may be empty.
3. `projects[].path` must be absolute after normalization.
4. `projects[].path` must resolve through symlinks (`EvalSymlinks`) successfully.
5. Canonical paths must be unique (no duplicates after canonicalization).
6. Optional `name` must be non-empty when provided.

### Config file write policy

1. Writes are atomic (`write temp file` then `rename`).
2. File permissions are user-readable/writable only (`0600`).
3. Project entries are persisted in deterministic order (stable by display name, then canonical path).

## Path Canonicalization Policy

Canonical path function for all comparisons:

1. `filepath.Abs`
2. `filepath.EvalSymlinks`
3. `filepath.Clean`

Matching rules:

1. Project match: exact canonical repo root equality.
2. Worktree association: canonical session pane path must equal or be within canonical worktree path.
3. No raw string prefix matching without separator checks.

If canonicalization fails for a configured project, the project is marked invalid in-memory and shown with an error indicator in TUI/list output.

## Discovery and Association Algorithm

For each configured project:

1. Canonicalize configured `project.path`.
2. Query `git -C <project> worktree list --porcelain`.
3. Parse all `worktree <path>` entries and canonicalize each path.
4. Keep entries equal to canonical project root or within canonical `<project>/.worktrees/`.
5. Build filesystem hierarchy: `project -> worktrees`, including a synthetic `(main repo)` node at canonical project root.

Then overlay tmux runtime:

1. List managed sessions from tmux.
2. For each session, fetch pane working directory and canonicalize.
3. Determine owning configured project by repo root canonical equality.
4. Determine owning worktree by longest canonical worktree-path match.
5. Attach session/windows/status under matched worktree.
6. If no `.worktrees` match but pane path is within canonical project root, attach under `(main repo)`.
7. Sessions that do not map to configured projects are excluded.

## Command Surface Changes

### New command group

1. `cb project add <path> [--name <display>]`
2. `cb project remove <path>`
3. `cb project remove --name <display>`
4. `cb project list`

### Behavior

1. `add` canonicalizes input and persists canonical path.
2. `remove` by positional argument requires exact canonical-path match after canonicalization (no fuzzy matching).
3. `remove --name` is explicit; it must match exactly one configured project name or return an ambiguity/not-found error.
4. `list` displays configured project names, canonical paths, and validation status.

### Existing commands

1. `cb dash`: project-scoped hierarchy as specified above.
2. `cb list`: same project filter and worktree hierarchy.
3. `cb start`: unchanged workflow creation, but if the current repo is not configured it prints a warning that new sessions will not appear in `cb dash` / `cb list`.
4. `cb clist`: unchanged and intentionally outside configured-project scoping.

## Data Model (proposed)

```go
type UserConfig struct {
    Version  int             `toml:"version"`
    Projects []ProjectConfig `toml:"projects"`
}

type ProjectConfig struct {
    Path string `toml:"path"`
    Name string `toml:"name,omitempty"`
}

type ProjectNode struct {
    Name      string
    Path      string // canonical project path
    Worktrees []WorktreeNode
    Invalid   error
}

type WorktreeNode struct {
    Name       string
    Path       string // canonical worktree path
    IsMainRepo bool   // true for synthetic "(main repo)" node
    Sessions   []SessionNode
}

type SessionNode struct {
    Name     string
    Status   tmux.Status
    Windows  []tmux.Window
    Expanded bool
}
```

## Error Handling and UX

1. Missing config file: show actionable message (`cb project add <path>`), then empty project-scoped view.
2. Empty projects list: valid; show actionable setup message and no project rows.
3. Invalid project path (missing/non-canonicalizable): keep node visible with error badge; do not crash.
4. Missing/empty `.worktrees` entries: show project with `(main repo)` node and hint to use `cb start`.
5. Session path inside project root but outside discovered `.worktrees` worktrees: show under `(main repo)`.
6. tmux unavailable/no sessions: keep project/worktree tree visible with inactive state.

## Backward Compatibility and Migration

1. No automatic migration is required for existing users with no project config.
2. There is no legacy fallback discovery mode for `cb dash` / `cb list`.
3. `cb dash` / `cb list` are authoritative project-scoped views immediately when this feature ships.
4. README/INSTALL updates are required for new `cb project` commands and behavior changes.

## Implementation Plan

### Phase 1: Config foundation

1. Add `config.toml` read/write support in `internal/config`.
2. Add project path canonicalization/validation utilities.
3. Implement atomic config writes with strict file permissions.
4. Add unit tests for parsing, validation, canonical uniqueness, and write behavior.

### Phase 2: Project commands

1. Add `cmd/project.go` with `add/remove/list`.
2. Wire command registration in root command.
3. Enforce remove-by-path default and explicit `--name` removal semantics.
4. Add command tests for happy path, invalid path, ambiguity, and empty-config behavior.

### Phase 3: Filesystem hierarchy + tmux overlay

1. Add a shared discovery layer for configured projects + worktrees + tmux session overlay.
2. Source worktree paths from `git worktree list --porcelain` parsing (including nested slash-branch layouts).
3. Add synthetic `(main repo)` worktree nodes and mapping behavior.
4. Refactor TUI fetch pipeline to consume the shared discovery model and produce `Project -> Worktree -> Session -> Window`.
5. Preserve existing status rollup and attach semantics.
6. Add/adjust TUI model/view tests.

### Phase 4: `cb list` alignment

1. Rework list output to use the same shared project/worktree-scoped discovery model.
2. Add deterministic ordering in output.
3. Add list command tests for grouping, filtered visibility, root-session mapping, and ordering.

### Phase 5: start warning + docs/polish

1. Add `cb start` warning when current repo is not configured.
2. Keep `cb clist` behavior unchanged.
3. Update `README.md` and `INSTALL.md` command/config docs.
4. Add troubleshooting for canonical path and symlink issues.

## Test Strategy

1. Config tests:
   1. valid TOML parse
   2. duplicate canonical path rejection
   3. symlink canonicalization
   4. invalid/missing path handling
   5. atomic write behavior and file mode
   6. deterministic serialization ordering
2. Discovery tests:
   1. `git worktree list --porcelain` parsing
   2. slash-branch nested worktree path support
   3. synthetic `(main repo)` node behavior
   4. longest-path worktree association
   5. mapping project-root sessions outside `.worktrees`
   6. exclusion of non-configured sessions
3. TUI tests:
   1. four-level hierarchy node flattening/navigation
   2. inactive worktree display
   3. `(main repo)` worktree rendering and navigation
4. List tests:
   1. project/worktree output grouping
   2. filtering behavior parity with TUI
   3. deterministic ordering
5. Start tests:
   1. warning displayed when starting from a repo not present in configured projects

## Acceptance Criteria

1. When two projects are configured, only those projects appear in `cb dash` and `cb list`.
2. Worktrees under `<project>/.worktrees/` appear beneath the correct project.
3. Sessions running from configured project root (outside `.worktrees`) appear under `(main repo)`.
4. Sessions are attached only under configured projects and mapped to the right worktree.
5. Symlinked project inputs resolve to canonical paths and are deduplicated.
6. `cb list` and `cb dash` produce consistent project/worktree scoping.
7. `cb project remove` requires canonical path by default; `--name` is explicit and unambiguous.
8. Empty config is valid and produces setup guidance instead of failure.
9. `cb start` warns when launched from an unconfigured repo.
10. No legacy fallback discovery mode is used by `cb dash` or `cb list`.

## Resolved Review Decisions (2026-02-14)

1. Inactive worktrees remain visible.
2. `cb project remove` defaults to exact canonical path matching.
3. Invalid configured projects remain warning-only.
4. Sessions outside `.worktrees` but inside configured project roots must be shown under `(main repo)`.
5. Empty project config is valid.
6. `cb start` warns when the repo is not configured.
7. `cb clist` remains unchanged.
8. No fallback discovery mode.
