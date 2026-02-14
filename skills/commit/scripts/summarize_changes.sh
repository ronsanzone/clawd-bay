#!/usr/bin/env bash
set -euo pipefail

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Not inside a git repository." >&2
  exit 1
fi

branch="$(git branch --show-current || true)"
if [[ -z "${branch}" ]]; then
  echo "Detached HEAD: resolve branch state before committing." >&2
  exit 1
fi

if ! git diff --quiet --check; then
  echo "Warning: whitespace issues or conflict markers detected by git diff --check." >&2
fi

echo "# Commit Change Summary"
echo
echo "- Branch: ${branch}"
echo "- Repo: $(git rev-parse --show-toplevel)"
echo
echo "## Status"
git status --short || true
echo
echo "## Diffstat (working tree + staged)"
git diff --stat
git diff --cached --stat
echo
echo "## Candidate Commit Buckets (top-level path)"

mapfile -t changed_files < <(git status --porcelain | awk '{print $2}' | sed 's#^"##;s#"$##' | sort -u)

if [[ ${#changed_files[@]} -eq 0 ]]; then
  echo "No uncommitted changes."
  exit 0
fi

declare -A buckets
for f in "${changed_files[@]}"; do
  top="${f%%/*}"
  if [[ "${f}" != *"/"* ]]; then
    top="(repo-root)"
  fi
  buckets["$top"]+="${f}"$'\n'
done

for key in "${!buckets[@]}"; do
  echo
  echo "### ${key}"
  while IFS= read -r line; do
    [[ -z "${line}" ]] && continue
    echo "- ${line}"
  done <<< "${buckets[$key]}"
done

echo
echo "## Recent Commits"
git log --oneline --decorate -n 8 || true
