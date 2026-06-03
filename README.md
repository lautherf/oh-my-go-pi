# omp — oh-my-pi (Go 移植版)

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/oh-my-pi/omp)](https://goreportcard.com/report/github.com/oh-my-pi/omp)

**omp** 是 [oh-my-pi](https://github.com/can1357/oh-my-pi) 的 **纯 Go 重写版**。原项目是 TypeScript + Rust 混合架构的 AI coding agent（~15 万行），本移植将其压缩为单一静态 Go 二进制，零 Node/Bun/Python 运行时依赖。

---

## 设计目标

- **单二进制**：`CGO_ENABLED=0` 静态编译，一个文件部署
- **零运行时依赖**：不需要 Node、Bun、Python、或任何系统包管理器
- **TDD 驱动**：每个包先写测试再实现，测试覆盖全流程
- **模块化**：17 个独立 Go package，清晰的责任边界

## 项目结构

```
cmd/omp/main.go          ← cobra CLI 入口
pkg/
├── ai/                  LLM 多供应商 / SSE 流式 / Ollama
├── agent/               Agent 循环 / 工具调用 / 状态管理
├── config/              viper YAML 配置
├── grep/                递归内容搜索（glob / 正则）
├── hashline/            Patch 引擎（锚定 diff）
├── jsonrpc/             JSON-RPC 2.0 Content-Length 帧协议
├── lsp/                 LSP 客户端 + Manager + Handler
├── mcp/                 MCP 客户端（JSON-RPC 工具调用）
├── memory/              SQLite 记忆存储
├── plugin/              扩展系统（HTTP 直连 npm registry）
├── safefs/              安全文件读写
├── shell/               os/exec + creack/pty
├── stats/               用户行为分析 + REST API
├── tools/               Agent 工具集
├── tui/                 bubbletea 差分渲染 TUI
├── utils/               日志 / 缓存 / 路径工具
└── web/                 网页搜索 + HTTP 抓取
internal/native/         token 计数 / 剪贴板
```

## 快速开始

```sh
# 构建
go build -o omp ./cmd/omp

# 交互模式（需要运行中的 Ollama）
./omp

# 使用指定模型
./omp --model llama3.2

# 插件管理
./omp plugin list
./omp plugin install <npm-package>

# 统计面板
./omp stats serve

# 性能调优
./omp --pprof :6060
```

## 与 TypeScript 版的差异

| 方面 | TypeScript 版 | Go 版 |
| --- | --- | --- |
| 运行时 | Node/Bun | 无（静态二进制） |
| 核心语言 | TS + Rust (~15 万行) | Go (~2 万行) |
| 交叉编译 | 平台特定构建 | goreleaser (linux/darwin/windows × amd64/arm64) |
| 安装 | npm / curl 脚本 | 单二进制下载 |
| 插件安装 | npm install | HTTP 直连 npm registry |
| LSP | Rust (tree-sitter) | Go (jsonrpc + client) |
| 文件系统隔离 (iso) | Rust (OverlayFS/APFS) | **不移植** |
| AST 操作 | tree-sitter | 待实现 |
| 调试器 (DAP) | Rust | 待实现 |
| 浏览器自动化 | Puppeteer | 待实现 |

## 测试

```sh
go test ./... -count=1
```

~110 个测试覆盖所有包，仅 `TestClipboard_CopyPaste` 在无 xclip/wl-clipboard 的环境中失败。

## 构建

```sh
# 开发
go build -o omp ./cmd/omp

# 交叉编译所有平台
goreleaser release --snapshot --clean
```

## 许可

MIT. 源自 [oh-my-pi](https://github.com/can1357/oh-my-pi) © Can Bölük，最初 fork 自 [Pi](https://github.com/badlogic/pi-mono) © Mario Zechner。
