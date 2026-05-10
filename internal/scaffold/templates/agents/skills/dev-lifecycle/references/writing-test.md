# Phase 7: Write Tests

Add tests targeting high coverage. **100% is the target**, not a hard requirement — trivial getters, generated code,
and unreachable error paths may be excluded with an explicit comment explaining why. Reference
`docs/ai/{name}/testing.md` and success criteria from requirements/design docs.

1. **Search memory** for existing testing patterns or known flaky areas:
   `aikits memory search --query "<name> testing patterns"` and `aikits memory search --query "<name> test gotchas"`.
2. **Gather context** — feature name, changes summary, environment (backend/frontend/full-stack), existing test suites, flaky tests to avoid.
3. **Analyze** the testing template, success criteria, edge cases, available mocks/fixtures.
4. **Unit tests** — cover happy path, edge cases, error handling for each module. Highlight missing branches.
5. **Integration tests** — critical cross-component flows, setup/teardown, boundary/failure cases.
6. **Coverage** — run coverage tooling, identify gaps, suggest additional tests if < 100%.
7. **Update** `docs/ai/{name}/testing.md` with test file links and results.
8. **Store** reusable testing patterns or setup gotchas after the quality gate passes:
   `aikits memory store --title "<testing pattern title>" --content "<context, guidance, evidence, exceptions>" --tags "<feature,testing>" --scope "repo:<org/repo>"`

**Next**: Phase 8 (Code Review). If tests reveal design flaws → back to Phase 3.
