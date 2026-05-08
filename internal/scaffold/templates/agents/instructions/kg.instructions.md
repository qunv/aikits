# Copilot Instructions

Use `aikits kg` as the default code navigation tool in this repo.

## Workflow
1. Run `aikits kg status`.
2. If the graph is missing or stale, detect the repo language(s) first from files like
   `go.mod`, `pom.xml`, `build.gradle`, `*.go`, or `*.java`.
3. Initialize and index only as needed:
   - `aikits kg init`
   - `aikits kg index --jobs 8 [--lang go|java|go,java]`
4. Before reading files, start with:
   - `aikits kg query symbol <name-or-fqn>`
   - `aikits kg query callers <fqn> --depth 3 --max-nodes 100`
   - `aikits kg query callees <fqn> --depth 3 --max-nodes 100`
   - `aikits kg query impact <fqn> --depth 3 --max-nodes 100`
5. For interface-heavy or inheritance-heavy code, also use:
   - `aikits kg query impls <interface-fqn> --depth 3 --max-nodes 100`
   - `aikits kg query overrides <method-fqn> --depth 3 --max-nodes 100`
6. If callsites need semantic precision, only use resolve steps for languages actually present:
   - `aikits kg resolve --lang go --budget 1000`
   - `aikits kg resolve --lang java --budget 1000 --maven-download-deps`

## When to use each query

| Query     | Use when …                                                       |
|-----------|------------------------------------------------------------------|
| `symbol`  | Finding a type, function, or variable by name                    |
| `callers` | Tracing who calls a function (impact of changing its signature)  |
| `callees` | Listing what a function calls (understanding its dependencies)   |
| `impact`  | Estimating the blast radius of a change                          |
| `impls`   | Discovering all implementations of an interface                  |
| `overrides` | Finding all method overrides in a class hierarchy              |

## Rules
- Read only the files the graph shows are relevant.
- Use `aikits kg status` again after indexing or resolving if you need progress or coverage info.
- Fall back to ripgrep only if graph results are incomplete.
- Prefer `--depth 3 --max-nodes 100` as default query parameters; increase if the result is too narrow.
