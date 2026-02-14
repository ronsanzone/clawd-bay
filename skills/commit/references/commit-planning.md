# Commit Planning Heuristics

Use these rules to split changes into logical commits:

1. Keep one behavior change per commit.
2. Separate mechanical changes (formatting, renames, generated files) from functional changes.
3. Separate tests from implementation only if tests are independently meaningful.
4. Keep shared refactors in a preparatory commit before feature commits.
5. Prefer small commits that pass tests over one large commit.

Suggested commit title patterns:
- `feat(scope): <summary>`
- `fix(scope): <summary>`
- `refactor(scope): <summary>`
- `test(scope): <summary>`
- `docs(scope): <summary>`
