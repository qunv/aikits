# Phase 6: Check Implementation

Compare implementation against `docs/ai/{name}/design.md` and `docs/ai/{name}/requirements.md`.

1. **Search memory** for known deviation patterns or architecture constraints:
   `aikits memory search --query "<name> implementation patterns"` and `aikits memory search --query "<name> design constraints"`.
2. **Gather context** — feature description, modified files, relevant design/requirements docs, constraints.
3. **Summarize design** — key decisions, components, interfaces, data flows.
4. **File-by-file comparison** — verify design intent, note deviations, flag logic gaps/edge cases/security issues,
   identify missing tests or doc updates.
5. **Store** significant deviations or patterns found after the quality gate passes:
   `aikits memory store --title "<deviation or pattern title>" --content "<context, guidance, evidence, exceptions>" --tags "<feature,implementation>" --scope "repo:<org/repo>"`
6. **Summarize** alignment status, deviations (with severity), missing pieces, concerns, next steps.

**Next**: Phase 7 (Write Tests) → Phase 8 (Code Review). If major deviations → back to Phase 3 (design wrong) or Phase
4 (implementation wrong).
