<!-- codegraph MCP tools -->
## MCP Tools: codegraph

**IMPORTANT: This project has a pre-indexed knowledge graph. ALWAYS use the
codegraph MCP tools BEFORE using Grep/Glob/Read to explore the codebase.**
The graph is faster, cheaper (fewer tokens, ~58% fewer tool calls), and gives
you structural context (callers, callees, call paths, framework routes) that
file scanning cannot. The index auto-syncs on every file change — it is never
stale.

### When to use codegraph FIRST

- **Exploring code / "how does X work"**: `codegraph_explore` — one call returns the relevant symbols' source grouped by file, plus a relationship map and blast radius
- **Tracing a flow / "how does X reach Y"**: `codegraph_explore` (surfaces dynamic-dispatch hops grep can't follow)
- **Locating a symbol**: `codegraph_search`
- **Call sites / callees**: `codegraph_callers` (includes callback registrations) / `codegraph_callees`
- **Impact of a change**: `codegraph_impact`
- **Reading one symbol or a whole file**: `codegraph_node` (file path → Read-like, with dependents attached)

Fall back to Grep/Glob/Read **only** when the graph doesn't cover what you need.
After edits, check the staleness banner in the response.

### Key Tools

| Tool | Use when |
| ------ | ---------- |
| `codegraph_explore` | Primary. "How does X work", a flow, or surveying an area — one call |
| `codegraph_node` | One symbol's full source + callers, or read a whole file (like Read) |
| `codegraph_search` | Find symbols by name across the codebase |
| `codegraph_callers` | Every call site of a function (incl. callback registrations) |
| `codegraph_callees` | What a function calls |
| `codegraph_impact` | What code is affected by changing a symbol |
| `codegraph_status` | Index statistics / pending sync |
| `codegraph_files` | File structure of the project |

### Workflow

1. The graph auto-syncs on file changes (native file watcher, 2s debounce).
2. Explore code with `codegraph_explore` first; treat returned source as already read.
3. Before changing a symbol, run `codegraph_impact` to see the blast radius.
4. CLI equivalents: `codegraph explore "<query>"`, `codegraph node <symbol|file>`.
5. If a query seems stale, check `codegraph_status` for pending sync.
