# Phase 5: Update Planning

Reconcile `docs/ai/planning/feature-{name}.md` with actual progress. Run once per session after completing one or more
tasks — not after every individual task. Batch all task completions from the session before reconciling.

# Phase 5: Update Planning

Reconcile `docs/ai/planning/feature-{name}.md` with actual progress. Run once per session after completing one or more
tasks — not after every individual task. Batch all task completions from the session before reconciling.

> **Note:** Phase 4 now persists task status directly to the planning doc as each task completes. In a normal hot flow,
> this phase is a summary and cleanup pass — not a bulk write. The Scan step is a safety net for interrupted sessions
> or work done outside the skill.

**Two entry modes — choose the right one:**

- **Hot (from Phase 4):** planning doc is already up-to-date from Phase 4 persistence. Carry forward the session
  summary. Skip Scan. Focus on milestones, summary paragraph, and next-task suggestions.
- **Cold (no Phase 4 context):** session was interrupted, or work was done outside the skill. Run Scan first.

If continuing from Phase 4, carry forward existing context. Otherwise, ask for feature name and run **Scan** first.

### Cold-start only: Scan

Check the codebase for evidence of completed tasks before asking anything. For each unchecked task in the planning doc,
verify whether the expected artifact exists (file path, function/type name, CLI sub-command). Use `find` and `grep` as
ground truth — do not rely on the user's memory. This replaces asking "what did you complete?"

### Steps (both modes)

1. **Review** existing milestones, sequencing, dependencies, outstanding tasks.
2. **Reconcile** each task: mark status (done/in-progress/blocked/not started), note scope changes, record blockers,
   capture skipped or added tasks.
3. **Update** the planning doc with current status checklist (done, in-progress, blocked, newly discovered work).
4. **Suggest** next 2-3 actionable tasks, risky areas, coordination needed.
5. **Write summary** paragraph for the planning doc: progress, risks, upcoming focus, scope changes.

**Next**: If tasks remain → Phase 4 (Execute Plan). If all done → Phase 6 (Check Implementation).
