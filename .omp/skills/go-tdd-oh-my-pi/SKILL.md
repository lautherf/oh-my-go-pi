---
name: go-tdd-oh-my-pi
description: Go 重写 oh-my-pi（TS+Rust → Go 单二进制）的完整总结。记录目标架构、分阶段路线图、关键决策、包结构、测试覆盖。供后续维护和参考。
---

# go-tdd-oh-my-pi — 项目总结

**目标**: 将 oh-my-pi（~15 万行 TypeScript + Rust）用 Go 重写为单静态二进制，不依赖 Node/Bun/Python 运行时。

**方法论**: TDD-first — 先写测试（红），再实现（绿），无测试不写代码。

---

## 最终状态

| 指标 | 值 |
| --- | --- |
| Phases | 0–6 **全部完成** |
| Go 包数 | 17 个 (`ai`, `agent`, `tools`, `hashline`, `shell`, `grep`, `safefs`, `config`, `utils`, `web`, `mcp`, `memory`, `plugin`, `stats`, `tui`, `lsp`, `jsonrpc`) + `internal/native` |
| 测试总数 | ~110 个（包测试 + e2e） |
| 全部通过 | ✅ `go vet ./...` + `go test ./...` |
| 唯一已知失败 | `TestClipboard_CopyPaste` — 环境无 xclip/wl-clipboard，非代码问题 |
| 外部依赖 | **零** Node/Bun/Python ❌，纯 Go 静态编译 ✅ |

---

## 项目结构

```
oh-my-pi/
├── cmd/omp/main.go            ← cobra CLI 入口（含 plugin/stats/pprof 子命令）
├── .goreleaser.yaml           ← 交叉编译（linux/darwin/windows × amd64/arm64）
├── pkg/
│   ├── ai/                    LLM 多供应商 / SSE 流式 / Ollama / 注册表
│   ├── agent/                 Agent 循环 / 工具调用 / 消息历史 / 状态压缩
│   ├── config/                viper YAML 配置加载
│   ├── grep/                  递归内容搜索（glob / 正则 / 大小写）
│   ├── hashline/              Patch 引擎（锚定 diff =/+ 操作）
│   ├── jsonrpc/               JSON-RPC 2.0 Content-Length 帧协议（传输层）
│   ├── lsp/                   LSP 客户端（Client + Manager + Handler + Types，24 测试）
│   ├── mcp/                   MCP 客户端（JSON-RPC 工具调用，6 测试）
│   ├── memory/                SQLite LIKE 记忆存储（modernc.org/sqlite）
│   ├── plugin/                扩展系统（HTTP 直连 npm registry 下载，25 测试）
│   ├── safefs/                安全文件读写（原子写入 / 路径防护 / 大小限制）
│   ├── shell/                 os/exec + creack/pty 终端模拟
│   ├── stats/                 用户行为分析 / SQLite 存储 / REST API / CLI（25 测试）
│   ├── tools/                 Agent 工具（read/write/edit/bash/grep/lsp）
│   ├── tui/                   bubbletea 差分渲染 TUI
│   ├── utils/                 zerolog 日志 / ristretto 缓存 / 路径工具
│   └── web/                   网页搜索 + HTTP 抓取（注册为 tool）
└── internal/native/           token 计数 / 剪贴板
```

---

## 各阶段总结

### Phase 0 — 骨架
- cobra CLI 框架搭建，viper YAML 配置，zerolog 日志，ristretto 缓存
- `go.mod` + 依赖管理
- **交付**: `omp --help`

### Phase 1 — 基础设施
- `pkg/shell/` — os/exec + creack/pty，不需要移植原版 brush（Rust 版 PTY 封装）
- `pkg/grep/` — re2 正则搜索，支持 glob 文件过滤
- `internal/native/` — token 计数 + 剪贴板（纯 Go 实现，无需 CGO）

### Phase 2 — LLM + Agent
- `pkg/ai/` — Provider 接口 + 注册表，支持 Ollama（HTTP mock 测试）
- SSE 流式解析
- `pkg/agent/` — 工具调用循环 / 消息历史 / 状态压缩

### Phase 3 — 核心工具
- `pkg/tools/` — read/write/edit/bash/grep 工具注册
- `pkg/hashline/` — Patch 引擎，解析和应用锚定 diff

### Phase 4 — TUI
- `pkg/tui/` — bubbletea 差分渲染，markdown/editor/input/list 组件
- 交互式 session 模式

### Phase 5 — 高级功能
| 包 | 说明 |
| --- | --- |
| `pkg/memory/` | SQLite 记忆存储，注册为 agent tool |
| `pkg/mcp/` | JSON-RPC MCP 客户端，6 测试全通过（helper 从 shell 脚本改为 Go 编译程序解决 ID 不匹配） |
| `pkg/plugin/` | 扩展管理器（已移除 npm 依赖） |
| `pkg/lsp/` | LSP 客户端 + Manager + Handler（含 jsonrpc 传输层） |
| `pkg/jsonrpc/` | Content-Length 帧协议，被 lsp 使用 |
| `pkg/web/` | 网页搜索 + HTTP 抓取，注册为 tool |
| `pkg/stats/` | 用户行为分析 + SQLite + REST API + CLI 子命令 |
| `pkg/iso/` | **决定不移植** — 原版 Rust 涉及 OverlayFS/APFS 系统调用，Go 成本高无收益 |

### Phase 6 — 打磨
- **移除 Node/Bun 依赖**: `pkg/plugin/` 从 `exec.Command("npm", "install")` 改为 HTTP 直连 npm registry，下载 tarball 解压。支持 `RegistryClient` 接口注入，httptest 可 mock。
- **交叉编译**: `.goreleaser.yaml` — linux/darwin/windows × amd64/arm64，含 nfpm（deb/rpm/apk）+ brew。
- **pprof**: `omp --pprof :6060` 启动 pprof HTTP 服务器。
- **端到端测试**: 8 个 CLI 烟雾测试（help/version/plugin/stats/unknown/pprof/interactive）。
- **LSP 合并**: LSP 项目从独立仓库合并回主项目 `pkg/lsp/` + `pkg/jsonrpc/`。

---

## 关键决策

- **单体二进制**：无需 FFI，Go 静态编译 `CGO_ENABLED=0`
- **并发模型**：goroutine + channel + errgroup 流式管道
- **Shell**：os/exec + creack/pty，不移植 brush
- **SQLite**：`modernc.org/sqlite`（纯 Go），与 memory/stats 包共用
- **Plugin**：npm registry HTTP 直连，不再依赖本地 npm 安装
- **LSP**：同步阻塞读（去掉 `StartDispatcher` goroutine），简化生命周期
- **MCP 测试**：Go 编译的 helper binary，自动回显请求 ID 解决 ID 不匹配问题
- **剪贴板**：`atotto/clipboard`，回退到纯 Go 终端选择
- **iso 不移植**：Rust 系统调用（OverlayFS/APFS），Go 重写收益低

---

## 测试架构

```
包测试（~100 个）
├── pkg/ai/       — httptest mock Ollama 服务器
├── pkg/mcp/      — Go 编译的 MCP helper binary
├── pkg/lsp/      — Go 编译的 LSP helper binary，22 测试
├── pkg/plugin/   — httptest mock npm registry，25 测试
├── pkg/stats/    — SQLite 内存模式，25 测试
├── pkg/memory/   — SQLite 内存模式
├── pkg/web/     — httptest mock 搜索 + 页面
├── pkg/shell/   — creack/pty 终端模拟
├── pkg/jsonrpc/ — Content-Length 帧协议编解码，11 测试
├── pkg/tools/   — read/write/edit/bash/grep 组合
└── pkg/hashline/ — diff 解析 + 应用

CLI e2e 测试（8 个）
├── TestCLI_Help / Version / PluginHelp
├── TestCLI_PluginList / StatsHelp
├── TestCLI_UnknownCommand / PprofFlag
└── TestCLI_InteractiveFailsWithoutOllama
```

---

## 迁移适配流程（归档参考）

```
1. 读原版源码 -> 理解合约
2. 写 Go 测试 -> 红
3. 最小实现 -> 绿
4. 对照原版补遗漏
5. 基准对比
```

原版行为 = spec。Go 习惯重写，但通过同一组合约。

---

## 依赖清单

| 领域 | Go 库 |
| --- | --- |
| CLI框架 | `spf13/cobra` |
| 配置 | `spf13/viper` |
| TUI | `charmbracelet/bubbletea` + `bubbles` + `lipgloss` |
| 日志 | `rs/zerolog` |
| SQLite | `modernc.org/sqlite`（纯 Go，无 CGO） |
| PTY | `creack/pty` |
| 语法高亮 | `alecthomas/chroma/v2` |
| Markdown | `yuin/goldmark` |
| HTML | `golang.org/x/net/html` + `goquery` |
| Diff | `sergi/go-diff` |
| Glob | `gobwas/glob` |
| 缓存 | `dgraph-io/ristretto` |
| 剪贴板 | `atotto/clipboard` |
| 测试 | `stretchr/testify` |
| 发布 | `goreleaser` |
