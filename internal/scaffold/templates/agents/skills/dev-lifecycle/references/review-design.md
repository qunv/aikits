# Phase 3: Review Design

Review `docs/ai/{name}/design.md` for completeness and fit against requirements.

1. **Search memory** for relevant architecture patterns or past decisions:
   `aikits memory search --query "<name> architecture patterns"` and `aikits memory search --query "<name> design decisions"`.
2. **Cross-check against requirements** — read `docs/ai/{name}/requirements.md` and verify every goal, user
   story, and constraint has corresponding design coverage. Flag uncovered requirements.
3. **Review completeness** — architecture (mermaid diagram), components, technology choices, data models, API contracts,
   design trade-offs, non-functional requirements.
4. **Clarify and explore (loop until converged)**:
    - **Ask clarification questions** for every gap or misalignment between requirements and design. Do not just list
      issues — actively ask specific questions. Example: "Requirements mention offline support but design has no
      caching — should we add one?"
    - **Brainstorm and explore options** — For key architecture decisions, trade-offs, or areas with multiple viable
      approaches, proactively brainstorm alternatives. Present options with pros/cons and trade-offs. Don't just accept
      the first approach — challenge assumptions and surface creative alternatives.
    - **Repeat** — Clarifying answers may reveal new trade-offs worth exploring, and brainstorming may surface new
      questions. Continue looping until the user is satisfied with the chosen approach and no open questions remain.
    - **Backward trigger**: if 2 or more requirements have no design coverage AND cannot be resolved by amending the
      current design, stop and return to **Phase 2** to fix requirements first before revising the design.
5. **Update** the design doc with clarified decisions and chosen options.
6. **Store** clarified architecture decisions in memory after the quality gate passes:
   `aikits memory store --title "<architecture decision title>" --content "<context, rationale, trade-offs, exceptions>" --tags "<feature,architecture>" --scope "repo:<org/repo>"`
7. **Summarize** requirements coverage, completeness assessment, updates made, remaining gaps.

**Next**: Phase 4 (Execute Plan). If requirements gaps found → back to Phase 2. If design fundamentally wrong → revise
design and re-review.
