---
name: dev-lifecycle
description: Structured SDLC workflow with 8 phases — new requirement, review requirements, review design, execute plan, update planning, check implementation, write tests, code review. Use when the user wants to build a feature end-to-end, or run any individual phase.
---

# Dev Lifecycle

Sequential phases producing docs in `docs/ai/`. Flow: 1→2→3→4→(5 after each task)→6→7→8.

## Prerequisite

Before starting any phase, run `aikits lint` to verify the base `docs/ai/` structure exists and is valid.

If lint fails because project docs are not initialized, run `aikits init`, then rerun lint. Do not proceed until checks pass.

For a **new feature start** (Phase 1 or `/new-requirement`), apply the shared worktree setup in [references/worktree-setup.md](references/worktree-setup.md) before phase work. This setup is worktree-first by default and includes explicit no-worktree fallback, context verification, and dependency bootstrap.

## Phases

| # | Phase                | Reference                                                                | When                                             |
|---|----------------------|--------------------------------------------------------------------------|--------------------------------------------------|
| 1 | New Requirement      | [references/new-requirement.md](references/new-requirement.md)           | User wants to add a feature                      |
| 2 | Review Requirements  | [references/review-requirements.md](references/review-requirements.md)   | Requirements doc needs validation                |
| 3 | Review Design        | [references/review-design.md](references/review-design.md)               | Design doc needs validation against requirements |
| 4 | Execute Plan         | [references/execute-plan.md](references/execute-plan.md)                 | Ready to implement tasks from planning doc       |
| 5 | Update Planning      | [references/update-planning.md](references/update-planning.md)           | Batch-reconcile after a Phase 4 session ends     |
| 6 | Check Implementation | [references/check-implementation.md](references/check-implementation.md) | Verify code matches design                       |
| 7 | Write Tests          | [references/writing-test.md](references/writing-test.md)                 | Add test coverage (100% target)                  |
| 8 | Code Review          | [references/code-review.md](references/code-review.md)                   | Final pre-push review                            |

Load only the reference file for the current phase. For Phase 1, also load [references/worktree-setup.md](references/worktree-setup.md).

## Resuming Work

If the user wants to continue work on an existing feature:

1. Check branch and worktree state before phase work:
    - Branch check: `git branch --show-current`
    - Worktree check: `git worktree list`
2. Determine target context for `<feature-name>` (all `.worktrees/` paths are relative to the **project root** — the directory containing `.git`):
    - Prefer worktree `<project-root>/.worktrees/feature-<name>` when it exists.
    - Otherwise use branch `feature-<name>` in the current repository.
3. Before switching, explicitly confirm target with the user (branch or worktree path).
4. After user confirmation, switch to the confirmed context first:
    - Worktree: run phase commands with `workdir=<project-root>/.worktrees/feature-<name>`.
    - Branch: checkout `feature-<name>` in current repo.
5. After switching, run `aikits lint` in the active branch/worktree context.
6. Then run the phase detector using the installed skill directory (same resolution rule as reference docs), not a workspace-relative `skills/...` path:
    - Resolve `<skill-dir>` as the directory containing this `SKILL.md`.
    - Run `<skill-dir>/scripts/check-status.sh <feature-name>` (or `cd <skill-dir> && scripts/check-status.sh <feature-name>`).
      Use the suggested phase from this script based on doc state and planning progress.

## Backward Transitions

Not every phase moves forward. When a phase reveals problems, loop back:

- Phase 2 finds fundamental gaps → back to **Phase 1** to revise requirements
- Phase 3 finds requirements gaps → back to **Phase 2**; design doesn't fit → revise design in place
- Phase 6 finds major deviations → back to **Phase 3** (design wrong) or **Phase 4** (implementation wrong)
- Phase 7 tests reveal design flaws → back to **Phase 3**
- Phase 8 finds blocking issues → back to **Phase 4** (fix code) or **Phase 7** (add tests)

## Doc Convention

Feature docs: `docs/ai/{name}/{phase}.md` (read from `<skill-dir>/phases/{phase}/README.md`, preserve front matter). Keep `<name>` aligned with the worktree/branch name `feature-<name>`.

Phases: `requirements`, `design`, `planning`, `implementation`, `testing`.

## Memory Integration

Run these CLI commands at phase boundaries (see the `memory` skill for full options and quality gate):

**Search first** — before asking questions or starting work. Use results as context; only ask about uncovered gaps.

| Phase | Search queries |
|-------|----------------|
| 1 | `aikits memory search --query "feature <name> past decisions"` · `aikits memory search --query "<name> conventions"` |
| 2 | `aikits memory search --query "<name> requirements constraints"` · `aikits memory search --query "<name> conventions"` |
| 3 | `aikits memory search --query "<name> architecture patterns"` · `aikits memory search --query "<name> design decisions"` |
| 4 | `aikits memory search --query "<name> implementation patterns"` · `aikits memory search --query "<name> gotchas"` |
| 6 | `aikits memory search --query "<name> implementation patterns"` · `aikits memory search --query "<name> design constraints"` |
| 7 | `aikits memory search --query "<name> testing patterns"` · `aikits memory search --query "<name> test gotchas"` |
| 8 | `aikits memory search --query "<name> code review findings"` · `aikits memory search --query "<name> security pitfalls"` |

**Store after** — once the quality gate passes. Use the narrowest useful scope (`repo:<org/repo>` preferred).

```bash
aikits memory store \
  --title "<actionable title, 10-100 chars>" \
  --content "<context, guidance, evidence, exceptions>" \
  --tags "<feature,phase>" \
  --scope "repo:<org/repo>"
```

Store per phase: decisions and conventions (1–3) · implementation patterns and gotchas (4) · deviations found (6) · testing patterns (7) · blocking review findings (8).

## Red Flags and Rationalizations

| Rationalization                          | Why It's Wrong                                | Do Instead                          |
|------------------------------------------|-----------------------------------------------|-------------------------------------|
| "Skip to coding, requirements are clear" | Ambiguity hides in assumptions                | Run Phase 1-3 first                 |
| "Design hasn't changed, skip Phase 6"    | Code drifts from design during implementation | Check implementation against design |
| "Tests slow us down, ship first"         | Bugs in production are slower                 | Write tests in Phase 4 and 7        |
| "Just a small change, no review needed"  | Small changes cause big outages               | Phase 8 applies to all changes      |

## Rules

- Read existing `docs/ai/` before changes. Keep diffs minimal.
- Use mermaid diagrams for architecture visuals.
- After each phase, summarize output and suggest next phase.
- Apply the `verify` skill before completing Phase 4 tasks, Phase 6 checks, Phase 7 coverage claims, and Phase 8 review items. No phase transition without fresh evidence.
- In Phase 4, apply the `tdd` skill (write failing test first, then make it pass). If the `tdd` skill is unavailable, write the failing test manually before writing production code.