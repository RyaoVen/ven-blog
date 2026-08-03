# tools/deploy — ven-blog 跨平台部署工具

独立 Go module（`ven_hybird/tools/deploy`，不依赖主模块），单二进制跨平台编译。
bubbletea TUI 为主、子命令为辅：环境检测 → 配置 → 构建 → 进程管理（Windows/Linux）。

## 用法

在仓库根或 `tools/deploy/` 目录下运行（自动向上定位仓库根）：

| 命令 | 说明 |
| --- | --- |
| `deploy` | 启动 TUI 主界面 |
| `deploy check` | 环境检测：go/node/npm 版本、MySQL 3306、Node/Go 端口（默认 3000/8080，随配置）、.env.local |
| `deploy config` | 查看/生成 `.env.local`（无文件时交互问答；内部令牌回车自动生成强随机值） |
| `deploy config --set KEY=VALUE [--set K2=V2]` | 追加/覆盖配置项（保留已有键，校验 BLOG_MYSQL_DSN） |
| `deploy build` | Node（`npm ci` + `npm run build`）→ Go（`go build -o bin/`） |
| `deploy start` | 编排启动：Node 先起 → 等 `/pages` 就绪（30s）→ Go 后起 → 等 `/api/site` 就绪（15s） |
| `deploy stop` | 强杀停止（读取 `.deploy/*.pid`） |
| `deploy restart` | 停止后重新启动 |
| `deploy status` | 进程存活 + 端口状态 |
| `deploy logs [-n N]` | tail 日志（默认 100 行） |

```bash
# 构建本工具
cd tools/deploy && go build -o deploy .
```

## TUI

- 顶部状态面板：构建产物 / 运行状态（PID）/ 端口（3000、8080、3306）/ 配置
- 菜单：检测 / 配置 / 构建 / 启动 / 停止 / 重启 / 日志 / 退出
- 操作：`↑`/`↓`（或 `j`/`k`）选择，`Enter` 执行，`q` 或 `Ctrl+C` 退出
- 执行视图：实时输出子命令结果，运行中 `Ctrl+C` 中断，完成后按任意键返回
- 日志视图：`↑`/`↓` 滚动，`Tab` 切换 node/go，`r` 刷新，`q` 返回

## 交叉编译

```bash
cd tools/deploy
GOOS=linux GOARCH=amd64 go build -o deploy-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o deploy.exe .
GOOS=darwin GOARCH=arm64 go build -o deploy-darwin .
```

## 实现说明

- `core/`：纯逻辑（可单测）——`envfile.go`（.env.local 手写解析/序列化，KEY=VALUE、`#` 注释、引号剥离）、
  `config.go`（DSN 必填校验 + `VEN_INTERNAL_TOKEN` 必填且拒绝默认值（网关强制）+ 端口解析）、
  `detect.go`（环境检测）、`build.go`（构建编排）、`proc.go`（进程管理）、`logs.go`（tail）
- `tui/`：bubbletea 界面——`app.go` 主模型（状态面板 + 菜单）、`runview.go` 执行视图（io.Pipe 实时输出）、
  `logsview.go` 日志视图
- 环境变量来自 `.env.local`，注入 Node/Go 子进程（文件值覆盖系统环境）
- PID 写 `.deploy/node.pid`、`.deploy/go.pid`；stdout/stderr 重定向 `logs/node.log`、`logs/go.log`
- **端口可配**：Node 端口读 `VEN_NODE_PORT`（默认 3000，与 Node 侧 config.ts 同读）；
  Go 端口从 `VEN_LISTEN_ADDR` 解析（`:8080`/`0.0.0.0:8080` 均可，默认 8080）。
  start/stop/status/check 全链路一致，子进程环境自动带上配置
- **就绪等待**：Node——`GET /pages`（`X-Ven-Internal-Token`，配置的令牌）1s×30；
  Go——`GET /api/site` 1s×15（2xx/4xx 都算网关活着；DSN 错/MySQL 挂时 Go 秒退或起不来，
  会区分「进程秒退」与「未就绪」提示并指向 logs/go.log，同时提示 Node 如何一并停止）
- **进程 detach**：Windows 子进程带 `CREATE_NEW_PROCESS_GROUP|DETACHED_PROCESS`（`proc_sysattr_windows.go`）、
  POSIX 带 `Setpgid`（`proc_sysattr_unix.go`）——关闭终端/Ctrl+C 不会波及服务进程；
  stop 仍按 PID 强杀有效（Windows `TerminateProcess`，失败 `taskkill /T /F` 连进程树一起杀）
- Windows 进程管理：`tasklist` 判定存活（按 PID 精确匹配，无本地化差异）、
  `TerminateProcess`/`taskkill /T /F` 强杀；POSIX：`kill(pid, 0)` 探测 + SIGKILL

## 注意事项

- 假设在仓库内运行（仓库根或 `tools/deploy/` 均可，自动向上查找 `frame/go`、`frame/node`）
- `scripts/dev.sh` 仍可用（bash 专用）；本工具为其推荐替代：Windows 可用、含构建/配置向导/进程管理
- 构建执行 `npm ci`，会重建 `frame/node/node_modules`（耗时较长属正常）
- `Ctrl+C` 中断构建时，Windows 下 npm 的孙进程可能残留（已知限制），可 `taskkill /F /IM node.exe` 兜底清理
- `deploy config` 展示时对 DSN 密码/令牌/API Key 打码，不会打印明文

## 开发（提交前必跑）

```bash
cd tools/deploy
go build ./... && go vet ./... && go test ./...
```
