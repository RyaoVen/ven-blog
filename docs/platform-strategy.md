# 平台策略与 devflow 集成（VenBlogDocs 项目）

> 状态：设计/前期准备阶段
> 日期：2026-09-15

## 1. 平台定位（用户拍板）

| 平台 | 定位 | 仓库 |
|---|---|---|
| **GitCode** | **dev 平台**（国内网络流畅）：日常开发、issue/PR/看板、devflow 全链路 | `liaoyutianyuan/ven-blog`（已建，私有，**当前为空待镜像**） |
| **GitHub** | **发布/同步平台**：一个 part/mod 在 dev 侧完成后迁移同步 | `RyaoVen/ven-blog`（上游真相源，master 分支） |

**同步策略**：GitCode dev 完成（devflow-ship 合并）一个完整 part/mod 后，向 GitHub 发同步 PR（或直接 push 对应 squash 提交），GitHub 侧保持"发布流水账"式干净历史。

## 2. 当前状态盘点（截至 2026-09-15）

| 事项 | 状态 |
|---|---|
| devflow 7 skill（plan/issue/impl/review/pr/board/ship） | ✅ 已装入通用目录 `~/.agents/skills/`（跨项目可见） |
| GitCode 仓库 `liaoyutianyuan/ven-blog` | ✅ 已创建（私有，issues 开启），**空仓库** |
| GitHub → GitCode 镜像推送 | ⏸️ 按边界暂停（本轮只做设计与前期准备） |
| 本地工作区 `~/Documents/VenBlogDocs` | ✅ 设计工作区（本目录，未 git 化） |
| 只读源码研究副本 | `/tmp/venblog-src`（浅克隆，仅供设计参考，可随时删） |
| 设计文档 | ✅ unit-6（插件系统治理）、unit-7（docs 插件）、本文档 |

## 3. devflow 落地清单（实施阶段开箱执行）

1. **镜像**：`git clone --mirror https://github.com/RyaoVen/ven-blog.git` → push `https://gitcode.com/liaoyutianyuan/ven-blog.git`（token 经 GIT_ASKPASS，不落日志/磁盘）→ clone 到 `~/Documents/VenBlogDocs`（origin=GitCode，另加 `github` remote 备同步）。
2. **工具落地**：复制 `~/Documents/VenWarehouse/tools/devflow/`（gitcode.sh / templates / backlog）入新仓 `tools/devflow/`；`devflow.env` 用 `cp` 复制后仅 `sed` 改三行（**不读取、不回显 token**）：
   ```diff
   - GITCODE_REPO=VenWarehouse
   - GITCODE_MAIN=main
   + GITCODE_REPO=ven-blog
   + GITCODE_MAIN=master      # ⚠️ ven-blog 默认分支是 master，不是 main
   ```
   并把 `devflow.env` 加入 `.gitignore`（随首个 PR 提交）。
3. **doctor 验通**：`bash tools/devflow/gitcode.sh doctor`。
4. **里程碑**：`gitcode.sh milestone-create` 建 M0~M5（§5）。
5. **种子 issue**：按 `tools/devflow/backlog/SEED-ISSUES.md`（§5 清单入库存档）逐张走 `/devflow-issue` 受理（不脚本批量建）。
6. **设计文档入库**：本目录 `design/` 三份文档迁入仓库 `docs/agent-design/`（unit-6/unit-7）+ `docs/platform-strategy.md`，随首个 PR。

## 4. 工作流约定（ven-blog 既有红线 × devflow 叠加）

- 分支：`<type>/issue-<N>-<slug>`；提交 Conventional Commits（中文描述 + `(#N)`）；禁直推 master、禁 force push。
- 门禁：每次提交前 `cd frame/go && go build ./... && go vet ./... && go test ./...` + `cd frame/node && npm run typecheck && npm test`。
- devflow 五道门中，PR 描述确认、合并确认等人工操作**由 agent 直接控制电脑代办**（用户已授权）；**不可逆的 merge/ship 仍需用户当次显式确认**。
- 框架层（`hybrid/`、`internal/`、`frame/node/`）绝不私改；缺口走 `docs/framework-requests.md` 上游流程（本期新增需求 7：catch-all 路由）。

## 5. 里程碑与种子 issue 清单（入库存档用）

| 里程碑 | 种子 issue（标题） | 验收标准要点 |
|---|---|---|
| M0 治理启动 | `[infra] 仓库镜像与 devflow 落地` | doctor OK；镜像一致；设计文档三份入库 |
| M0 治理启动 | `[docs] 上游需求 7：catch-all 页面路由` | Node `[...slug]` 推导 + Go ISR 通配；或过渡方案三条固定深度声明 |
| M1 插件内核 | `[plugin] Unit6 插件契约与注册内核` | §8 验收：echo 测试插件全流程 + 全关 bit-exact |
| M1 插件内核 | `[plugin] MCP 扩展点（RegisterAction）` | 内置 14 action 零行为变更；extra 表注册/冲突拒绝 |
| M2 docs 后端 | `[docs] 数据模型与 docapp 用例` | 领域规则单测全绿 |
| M2 docs 后端 | `[docs] MCP doc.* 基础七 action` | 冒烟全绿；失效声明生效 |
| M3 docs 前端 | `[docs] /docs 页面（树导航/ISR/SSE）` | 页面 200；失效再生 |
| M3 docs 前端 | `[docs] admin 编辑器` | 后台全流程 |
| M4 hooks | `[docs] doc.move + since 增量 + webhook` | HMAC 验证 |
| M5 内容 | `[docs] import/export + 首批笔记导入` | MyNotes/Docs 选摘成树 |
| M6 搜索 | `[search] 搜索集成与 scope 选项（blog/docs/all）` | 三 scope 冒烟；不装 docs 插件 bit-exact |
| M5 内容 | `[infra] GitHub 同步流程演练` | 一个 mod 完整走完 dev→publish |

## 6. 边界备忘（本轮）

> 用户指示（2026-09-15）：**只做设计与前期准备，不做具体落地。**
> 已完成边界内事项：devflow skill 安装、GitCode 空仓创建、框架源码研究、三份设计文档。
> 明确未做（待指令）：镜像推送、工作区 git 化、里程碑/issue 平台创建、任何业务代码。
