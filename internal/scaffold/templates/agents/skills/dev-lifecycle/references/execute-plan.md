# Phase 4: Execute Plan

Work through `docs/ai/{name}/planning.md` one task at a time.

1. **Gather context** — feature name, planning doc path, supporting docs (design, requirements), current branch/diff.
2. **Load plan** — parse task lists (checkboxes), build ordered queue by section.
3. **Present task queue** with status: `todo`, `in-progress`, `done`, `blocked`.
4. **For each task**: show context, suggest relevant docs, offer to outline sub-steps from design doc. Apply the `tdd`
   skill — write a failing test before production code, then make it pass. If the `tdd` skill is unavailable, write the
   failing test manually first. If blocked, record blocker and defer.
5. **Persist task status** — after each task completes (or is blocked/skipped), immediately update the checkbox in
   `docs/ai/{name}/planning.md`: `[ ]` → `[x]` for done, `[ ]` → `[~]` for blocked (add blocker note inline).
   Do not wait until Phase 5. This makes progress resilient to session interruptions.
6. **After all sections are done**, ask once: "Were any new tasks discovered during this session?"
7. **Session summary** — completed, in-progress, blocked, skipped, new tasks.

**Next**: After the session ends → Phase 5 (Update Planning) to reconcile all changes at once. When all tasks done → Phase 6 → 7 → 8.
