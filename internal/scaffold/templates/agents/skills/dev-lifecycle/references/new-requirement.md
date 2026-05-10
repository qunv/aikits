# Phase 1: New Requirement

1. **Search AI DevKit memory** (not built-in memory) for relevant past features or conventions via
   `aikits memory search --query "feature <name> past decisions"` and `aikits memory search --query "<name> conventions"`.
   If unfamiliar, check the AI DevKit memory skill first.
2. **Ask** for: feature name (kebab-case), problem, target users, key user stories. Skip what memory already covers;
   store answers after. **Brainstorm**: ask clarifying questions as needed, explore alternatives to confirm this is the
   right thing to build, then present 2–3 approaches for the chosen direction — one-line trade-offs + recommendation.
3. **Run shared setup first** using [worktree-setup.md](worktree-setup.md) with normalized `<name>`:
    - Default: create and use `feature-<name>` worktree
    - Optional fallback: no-worktree only when user explicitly requests it
    - Required guards: context verification + dependency bootstrap
4. **Create docs** — for each phase (`requirements`, `design`, `planning`, `implementation`, `testing`), read
   `<skill-dir>/phases/{phase}/README.md` and copy it to `docs/ai/{name}/{phase}.md`. Preserve frontmatter. Leave
   design, planning, implementation, and testing docs as empty templates — they will be filled in later phases.
5. **Fill requirements doc** — problem statement, goals/non-goals, user stories, success criteria, constraints, open
   questions.

**Next**: Phase 2 (Review Requirements). Design and planning docs are filled after requirements are reviewed and approved in Phase 2→3.
