---
name: oh-my-pi
description: 原版 TS+Rust 项目参考。描述现有代码库结构、约定、命令。在维护/修改现有 Bun+TS 代码时加载。Go 迁移相关请加载 go-tdd-oh-my-pi。
---

# oh-my-pi Project

Monorepo managed by **Bun** (TypeScript packages) + **Cargo** (Rust crates). CLI binary: `omp`.

## Package Structure

| Package | Description |
| --- | --- |
| `packages/coding-agent` | Main CLI app (**primary focus**) |
| `packages/ai` | Multi-provider LLM client, streaming, model resolution |
| `packages/agent` | Agent runtime — tool calling, state management |
| `packages/tui` | Terminal UI library, differential rendering |
| `packages/natives` | TS bindings for native text/image/grep operations |
| `packages/utils` | Shared utilities — logger, streams, temp files |
| `packages/stats` | Local observability dashboard (`omp stats`) |
| `packages/hashline` | Patch format engine (edit DSL) |
| `packages/mnemopi` | Memory backend (mnemonic storage) |
| `crates/pi-natives` | Rust — perf-critical text/grep ops |
| `crates/pi-ast` | Rust — AST-level editing engine |
| `crates/pi-shell` | Rust — shell minimizer / minimizer defs |
| `crates/pi-iso` | Rust — filesystem diff, clone, overlay, snapshot |
| `crates/brush-core-vendored` | Rust — vendored brush shell core |
| `crates/brush-builtins-vendored` | Rust — vendored brush builtins |

## Key Commands

| Command | Action |
| --- | --- |
| `bun run dev` | Run omp CLI from source |
| `bun test` | Run all TS + Rust tests |
| `bun run check` | TypeScript typecheck + Biome lint + Rust check |
| `bun run lint` | Biome lint (TS) + Clippy (Rust) |
| `bun run fmt` | Biome format (TS) + Rustfmt |
| `bun run fix` | Auto-fix issues |
| `bun run build:native` | Build native Rust -> `.node` binary |
| `bun run generate-models` | Regenerate `packages/ai/src/models.json` |
| `bun run release` | Version bump, CHANGELOG, tag, publish |

## Code Conventions

- **No `any`** unless absolutely necessary.
- **NEVER use `ReturnType<>`** — use the actual type.
- **NEVER inline imports** — no `await import()`, no `import("pkg").Type`.
- **Class privacy**: ES `#private` fields; no `private`/`protected`/`public` keyword.
- **Promises**: use `Promise.withResolvers()`.
- **Prompts**: live in static `.md` files with Handlebars; import via `import content from "./prompt.md" with { type: "text" }`.
- **Bun over Node**: `Bun.file()`, `Bun.write()`, `Bun.spawn()`, `$`cmd``, `Bun.sleep()`.
- **Namespace imports** for `node:*` modules.
- **No `console.log`/`error`/`warn`** in coding-agent — use `logger` from `@oh-my-pi/pi-utils`.
- **NEVER edit `packages/ai/src/models.json`** directly.
- **No `tsc`/`npx tsc`** — always `bun check`.
- **NEVER commit unless asked.**

## Architecture (existing TS+Rust)

```
User Input -> CLI (cli.ts) -> main.ts -> sdk.ts -> createAgentSession()
    |
    v
Agent Loop (agent.ts) -> LLM Call (ai/stream.ts) -> Provider API
    |                           |
    |                           v
    |                       Tool Execution (tools/*.ts)
    |                           |--- BashTool --> native Rust brush shell
    |                           |--- ReadTool --> grep/glob (native Rust)
    |                           |--- EditTool --> hashline (TS)
    |                           |--- SearchTool --> web providers
    |                           |--- AstGrep/Edit --> native Rust pi-ast
    |                           +--- ... 60+ tools
    |
    v
Session persisted to SQLite / Result to TUI renderer
```

## Related Skills

| Skill | Purpose |
| --- | --- |
| `go-tdd` | 通用 Go TDD 方法论，写 Go 测试时加载 |
| `go-tdd-oh-my-pi` | Go 迁移执行计划与进度，做迁移时加载 |
