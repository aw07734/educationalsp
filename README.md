# educationalsp

一个用 Go 从零实现的 **LSP（Language Server Protocol）语言服务器**，用于跟着视频教程一步步理解 LSP 的工作原理。

> 配套教程：[**Learn By Building: Language Server Protocol**](https://www.youtube.com/watch?v=YsdlcQoHqPY) — TJ DeVries

它不是一个真正可用的语言服务器，而是一个**教学用的最小可运行实现**：没有真正的语法分析、没有 AST、没有类型推断。它演示的是 LSP 的**协议层**——消息怎么编码、请求怎么分发、结果怎么回给编辑器。

---

## 特性

服务器实现了以下 LSP 能力（详见 `lsp/initialize.go` 中的 `ServerCapabilities`）：

| 能力 | 触发方式 | 实现行为 |
|---|---|---|
| `initialize` | 客户端握手 | 返回服务器能力声明与 `serverInfo` |
| `textDocument/didOpen` | 打开文件 | 存入文档缓存，并推送诊断信息 |
| `textDocument/didChange` | 编辑文件 | 全量同步文本，重新推送诊断信息 |
| `textDocument/publishDiagnostics` | 服务端推送 | 在编辑器的文件上显示波浪线提示 |
| `textDocument/hover` | 鼠标悬停 | 返回文件 URI 与字符数 |
| `textDocument/definition` | 跳转到定义 | 返回上一行的起始位置 |
| `textDocument/codeAction` | 快速修复（`Ctrl+.`） | 提供「替换 / 屏蔽」两种修复动作 |
| `textDocument/completion` | 代码补全 | 提供一条补全建议 |
| `exit` | 退出 | 结束进程 |

### 这个服务器在「教」你什么

诊断规则只有一条：**如果文本里出现 `VS Code`，就报错**，提示 *"Please make sure we use good language in this video"*。

紧接着，代码修复动作会一本正经地提供两个选项：

- **Replace VS C\*de with a superior editor** → 把 `VS Code` 全量替换成 `Neovim`
- **Censor VS C\*de** → 把 `VS Code` 替换成 `VS C*de`

补全列表里也永远只有一条 `Neovim(BTW)`。

这是教程作者的一个玩笑（**BTW = Neovim 社区对 VS Code 的昵称**），但玩笑背后是完整的功能：**诊断 → 定位范围 → 生成 `WorkspaceEdit` → 客户端应用编辑** 这条链路，一个不少。

> 想验证效果，打开仓库根目录的 [`Test.md`](./Test.md) 试试——里面正好有 4 行 `VS Code` 和 1 行 `Neovim(BTW)`。

---

## 快速开始

### 环境要求

- Go **1.26.4**（见 `go.mod`）

本项目**零第三方依赖**，只用标准库，无需 `go mod tidy` 拉取任何包。

### 编译

```bash
go build -o educationalsp
```

在 Windows 上会生成 `educationalsp.exe`。

### 运行

LSP 服务器通过 **stdin / stdout** 与编辑器通信，所以它必须由编辑器拉起，**不要**在终端里直接运行它（直接运行会因等待 stdin 而挂起）。

```bash
go run .
```

运行日志会写入 `output.log`，路径由 `logPath()` 通过 `os.Executable()` 解析，**始终位于可执行文件所在目录**，不随工作目录变化。若无法确定可执行文件路径，则回退为相对路径（基于当前工作目录）。

> 💡 用 `go run .` 时，Go 会把临时可执行文件放在系统临时目录，因此日志也会落在那里。要在项目目录看到 `output.log`，请先 `go build` 再直接运行编译产物。

### 在编辑器中接入

以 VS Code 为例，新建一个扩展目录并用 `Code --extensionDevelopmentPath=<目录>` 启动，扩展的激活事件里指向本项目编译出的可执行文件即可（具体配置参见视频教程对应章节）。

Neovim 用户可以直接用内置的 `vim.lsp.start()`，把 `cmd` 指向编译产物。

---

## 项目结构

```
.
├── main.go           # 入口：stdin 扫描 → 消息分发 → stdout 写回
├── analysis/         # 业务层：文档缓存 + 具体的分析逻辑
│   └── state.go      # State 结构体，所有 LSP 能力的实现都在这里
├── lsp/              # 协议层：纯数据结构，定义请求/响应的 JSON 形状
│   ├── message.go    # Request / Response / Notification 基类
│   ├── initialize.go # 握手与能力声明
│   ├── textdocument*.go # 各能力对应的类型定义
│   └── textdocument.go   # Position / Range / TextEdit 等公共类型
├── rpc/              # 传输层：JSON-RPC 的帧编解码
│   ├── rpc.go        # EncodeMessage / DecodeMessage / Split
│   └── rpc_test.go   # 唯一的单元测试
├── Test.md           # 用于手工验证诊断与代码动作的测试文件
└── output.log        # 运行日志（已被 .gitignore 忽略）
```

三层职责划分是理解这个项目最关键的一点：

```
编辑器  ──stdin/stdout──▶  main.go  ──▶  rpc（拆帧/编解码）
                                          │
                                          ▼
                                      lsp（类型定义）
                                          │
                                          ▼
                                    analysis（业务逻辑）
```

- **`rpc`** 只关心字节流：把 `Content-Length: N\r\n\r\n{...}` 切开、取出 JSON。它**不认识 LSP**。
- **`lsp`** 只关心 JSON 的形状：定义好结构体和 `json` tag，**没有逻辑**。
- **`analysis`** 只关心业务：拿到文档文本，做出判断，把结果交给 `lsp` 的类型装好。

---

## 实现要点

### JSON-RPC 帧格式

LSP 的消息不是普通 JSON 行，而是带长度前缀的帧：

```
Content-Length: 16\r\n\r\n{"testing":true}
```

`rpc.Split` 被直接交给 `bufio.Scanner.Split` 作为切分函数（见 `main.go:19`），这是整个服务器最巧妙的一处设计：**利用 Scanner 已有的缓冲机制处理半包**，不需要自己写读循环。

### 区分 Request、Notification 与 Response

- **Request**：客户端发起，带 `id`，服务器必须回一个同 `id` 的 Response。
- **Notification**：单向通知，**不带 `id`**（如 `didOpen`、`publishDiagnostics`）。
- **Response**：`ID` 字段是 `*int`（见 `lsp/message.go:14`），因为 Notification 不需要 id，序列化时会自动省略。

### 诊断与代码动作的联动

`analysis/state.go:129` 的 `LineRange` 是贯穿诊断与修复的关键工具——诊断用它把 `VS Code` 标红，代码动作用它生成精确的 `TextEdit`。两者必须指向**完全相同的范围**，编辑器才能把波浪线和快速修复对应起来。

---

## 测试

```bash
go test ./...
```

当前只有 `rpc` 包有单元测试（覆盖编码与解码的往返），`main` 和 `analysis` 包暂无测试文件。给 `analysis` 补测试是个不错的练习方向。

预期输出：

```
?   	educationalsp	[no test files]
?   	educationalsp/analysis	[no test files]
?   	educationalsp/lsp	[no test files]
ok  	educationalsp/rpc
```

---

## 已知限制

这是一个教学项目，以下都是**有意为之**的简化：

- 无语法解析、无 AST、无类型系统
- 文档同步是 `Full`（全量）模式，未实现增量同步
- `definition` 不查符号表，只返回固定位置
- `didClose`、`shutdown` 等方法只记录日志，未做实际处理
- 仅支持单一客户端，未做多会话隔离

把它变成真正有用的语言服务器，正是这个教程后续章节的方向。

---

## 相关链接

- [Language Server Protocol 规范](https://microsoft.github.io/language-server-protocol/)
- [vscode-languageserver-node](https://github.com/microsoft/vscode-languageserver-node) — 生产级 LSP 的参考实现
- 配套视频：[Learn By Building: Language Server Protocol](https://www.youtube.com/watch?v=YsdlcQoHqPY)

---

## 许可证

本项目基于 [Apache License 2.0](./LICENSE) 开源。
