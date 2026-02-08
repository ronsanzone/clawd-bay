# PR Review Improvements Design

**Date:** 2026-02-02
**Status:** Draft
**Source:** Analysis of quick-review skill for features to port into pr-review

## Overview

This design documents improvements to the `pr-review` skill based on analysis of the `quick-review` skill. The goal is to improve coverage (catch more real issues) and output quality (make findings more actionable).

## Changes Summary

| Change | Type | Description |
|--------|------|-------------|
| Agent #2 expansion | Coverage | Add error handling checks to Shallow Bug Scan |
| Agent #8 (new) | Coverage | Correctness Validation — verify PR solves stated problem |
| Executive Summary | Output | Add review-level assessment separate from PR summary |
| Code fix examples | Output | Add concrete code suggestions to each issue |

## Detailed Design

### 1. Agent #2: Bug & Error Handling Scan

**Current description:**
```
Agent #2: Shallow Bug Scan
Read file changes and scan for obvious bugs. Focus on the diff only,
not surrounding context. Target large bugs, avoid nitpicks. Ignore
likely false positives.
```

**New description:**
```
Agent #2: Bug & Error Handling Scan
Read file changes and scan for:

**Bugs:**
- Logic errors and off-by-one mistakes
- Null/undefined dereferences
- Race conditions in concurrent code
- Edge cases not handled

**Error Handling:**
- Missing try/catch around operations that can fail
- Silent failures (caught but not logged/handled)
- Unvalidated user input before use
- Missing null checks before dereference
- Error messages that leak sensitive information
- Failure recovery paths that leave inconsistent state

Focus on the diff only, not surrounding context. Target significant
issues, avoid nitpicks. Ignore likely false positives.
```

**Rationale:** Error handling is closely related to bug detection — a missing null check is both a bug and an error handling gap. Keeping them together avoids duplicate analysis of the same code.

---

### 2. Agent #8: Correctness Validation (New)

**Add to Step 4 (Parallel Review Agents):**

```
Agent #8: Correctness Validation
Verify that the code changes actually solve the stated problem.

**Inputs:**
- PR title and description
- Jira ticket details (if linked in PR description or branch name)
- The diff

**Process:**
1. Extract the stated intent from PR description
2. If Jira ticket linked, fetch ticket summary and acceptance criteria
3. Analyze whether the code changes address the stated problem

**Flag issues when:**
- PR claims to fix X, but the fix doesn't address the root cause
- PR claims to add feature Y, but implementation is incomplete
- Jira acceptance criteria exist but aren't met by the changes
- PR description is vague/missing and changes are non-trivial

**Do NOT flag:**
- PRs with clear description that match the implementation
- Refactoring PRs where "correctness" is subjective
- Trivial changes (typo fixes, version bumps)
```

**Jira integration:** The agent uses the `jira-cli` skill (if available) to fetch ticket details when a Jira key is detected in the PR description or branch name (e.g., `PROJ-123`). This is optional — the agent works without Jira by validating against the PR description alone.

**Model:** Opus (matches other review agents)

**Pipeline impact:** 7 agents → 8 agents. New agent runs in parallel with existing agents, so no latency increase.

---

### 3. Output Format: Executive Summary

**Current structure in Step 8:**
```markdown
### Summary
<2-3 sentence summary from Step 3>  ← This summarizes the PR, not the review
```

**New structure:**
```markdown
### Summary
<2-3 sentence summary of what the PR does>

### Executive Summary
<2-3 sentence assessment of the review findings>

Examples:
- "Solid implementation with one critical auth vulnerability that must
   be addressed. Two medium-priority issues around error handling."
- "Clean PR with no significant issues. Minor style suggestions only."
- "Several correctness concerns — the fix doesn't appear to address
   the root cause described in PROJ-456."
```

**Rationale:** The PR summary tells you what the PR does. The Executive Summary tells you what the review found. Both are valuable for quickly understanding the situation.

---

### 4. Output Format: Code Fix Examples

**Current issue format:**
```markdown
1. **<description>** (Source: <agent>, Score: <N>)
   - File: `path/to/file.go:123`
   - Reason: <CLAUDE.md says "..." | bug due to ... | security: ...>
   - Link: <GitHub link with full SHA>
```

**New issue format:**
```markdown
1. **<description>** (Source: <agent>, Score: <N>)
   - File: `path/to/file.go:123`
   - Reason: <why this is a problem>
   - Suggested fix:
     ```go
     // concrete code example showing the fix
     ```
   - Link: <GitHub link with full SHA>
```

**Guidance for agents:** Code examples are best-effort. If the fix is architectural or context-dependent, describe the approach instead of providing literal code. The goal is actionability — the reviewer should know exactly what to do.

---

## Updated Quick Reference

| Component | Model | Purpose |
|-----------|-------|---------|
| Eligibility | Haiku | Gate: skip drafts/closed/automated |
| CLAUDE.md finder | Haiku | Locate project standards |
| Summarizer | Haiku | PR overview |
| Reviewers (8x) | Opus | Deep analysis |
| Scorers (per issue) | Haiku | Confidence calibration |

| Agent | Focus |
|-------|-------|
| #1 | CLAUDE.md Compliance |
| #2 | Bug & Error Handling Scan |
| #3 | Git History Context |
| #4 | Previous PR Comments |
| #5 | Code Comment Compliance |
| #6 | Security Analysis |
| #7 | Test Coverage Check |
| #8 | Correctness Validation |

## Implementation Notes

1. Update `pr-review/skill.md` with the changes above
2. Test with a few PRs to validate:
   - Error handling issues are caught by Agent #2
   - Correctness validation works with and without Jira
   - Executive Summary accurately reflects findings
   - Code fix examples are actionable

## Origin

These improvements were identified by comparing `quick-review` (single-pass expert review) with `pr-review` (multi-agent ensemble review). The quick-review skill had:

- Explicit error handling in its Analysis Framework
- "Why it matters" explanations (influenced the code fix examples)
- Executive Summary of review findings
- Concrete code suggestions for each issue

The goal is to bring these strengths into pr-review while maintaining its multi-agent architecture and confidence scoring.
