---
name: go-tdd
description: 通用 Go TDD 方法论。不绑定具体项目。红绿重构循环、测试约定、mock/assert/benchmark 最佳实践。在任何 Go 项目中写测试时加载。
---

# Go TDD 方法论

**红-绿-重构** 循环驱动每段代码。不允许「先实现再补测试」。

## Workflow

```
1. 写测试（红） -- 定义合约，编译通过但运行失败
2. 写实现（绿） -- 最小代码让测试通过
3. 重构     -- 改进结构，测试保持绿色
4. 重复
```

## 黄金法则

1. **红色先于绿色**：任何 `*.go` 文件写入前，必须有对应 `*_test.go` 定义合约
2. **最小实现**：只够让当前测试通过，不超前设计
3. **一次一个失败**：一次只让一个测试从红变绿
4. **重构时不改行为**：重构阶段只改结构，不改测试
5. **测试即文档**：测试命名说明功能，断言说明边界

## 测试约定

### 文件布局
```
pkg/ai/
+-- stream.go
+-- stream_test.go              # 单元测试（同一包）
+-- provider_test.go             # 集成测试（mock server）
+-- provider_integration_test.go # [integration] build tag，真实 API
```

### 命名
- `TestXxx_Yy` -- 驼峰描述场景
- `TestXxx_TableDriven` -- 表格驱动测试
- `BenchmarkXxx` -- 基准测试

### 结构
```go
func TestSomething_WithDescription(t *testing.T) {
    t.Parallel()
    // arrange
    // act
    // assert
}
```

### 工具链

| 工具 | 用途 |
| --- | --- |
| `testing` (stdlib) | 基础框架，基准测试，子测试 |
| `github.com/stretchr/testify` | assert/require 断言 |
| `go.uber.org/mock` | mock 生成 (`go generate`) |
| `testcontainers-go` | 容器化集成测试 |
| `testing/quick` | 属性/模糊测试 |
| `go test -race` | 竞态检测（CI 必须开启） |
| `go test -coverprofile` | 覆盖率报告 |

### 覆盖率目标
- 核心逻辑: >= 85%
- 胶水/配置代码: >= 70%
- UI 组件: >= 60%

## 迁移适配（通用）

从其他语言移植模块时:

```
1. 读透原版，理解合约（输入/输出/错误/边界）
2. 写 Go 测试覆盖所有边界（红）
3. 实现最小功能让测试通过（绿）
4. 对照原版补遗漏场景
5. 基准测试对比性能
```

原版行为 = spec，不是实现。以 Go 习惯重写，但通过同一组合约测试。

## 典型循环

```go
// RED
func TestAdd(t *testing.T) {
    assert.Equal(t, 4, Add(2, 2))
}

// GREEN
func Add(a, b int) int { return a + b }

// REFACTOR
func Add(nums ...int) int {
    s := 0
    for _, n := range nums { s += n }
    return s
}
```

## 禁止

- ❌ 先写 `impl.go` 再补 `impl_test.go`
- ❌ 测试用 `fmt.Println`（用 `t.Log`）
- ❌ 依赖未实现模块（用 mock 隔离）
- ❌ 外部网络测试不加 `-short` / build tag
- ❌ 测试间共享可变状态
- ❌ 覆盖率不达标的 PR 合入
