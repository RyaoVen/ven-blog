# VenDevFlow 工具集

驱动开发工作流（docs/09-dev-workflow.md）的落地面：`.agents/skills/devflow-*` 提供流程指令，本目录提供被 skill 调用的脚手架。

```
tools/devflow/
├── gitcode.sh              # GitCode OpenAPI 封装（curl + jq）
├── devflow.env.example     # 配置模板 → 复制为 devflow.env（勿入库）
└── templates/
    ├── ISSUE_TEMPLATE.md   # /devflow-issue 渲染用
    ├── PLAN_TEMPLATE.md    # /devflow-plan 渲染用
    ├── PR_TEMPLATE.md      # /devflow-pr 渲染用
    └── REVIEW_CHECKLIST.md # /devflow-review 检查单
```

## 快速开始

```bash
cp devflow.env.example devflow.env   # 填入 OWNER/REPO/TOKEN（经典 PAT）
./gitcode.sh doctor                  # 验证令牌与仓库
./gitcode.sh help                    # 查看全部子命令
```

## 常用子命令

| 命令 | 说明 |
|---|---|
| `issue-create <title> <body(-\|@file)> [labels] [assignee]` | 建工单（body 用 `@文件` 或 `-` 读 stdin） |
| `issue-list [state] [labels]` / `issue-get <n>` | 工单查询 |
| `issue-comment <n> <body>` | 工单评论（贴方案/评审/进度） |
| `pr-create <title> <head> <base> <body>` | 建 PR |
| `pr-link <pr> <issue...>` | PR 关联工单（合并后联动关闭） |
| `pr-merge <n> [merge_method]` | 合并（默认 squash） |
| `milestone-list [state]` / `milestone-create <title> <desc> [due_on]` | 里程碑管理（基线见 docs/10） |
| `branch-create <name> [ref]` | 建远端分支 |

一次性引导：`./bootstrap.sh --dry-run` 预览后 `./bootstrap.sh` 创建 M0~M5 里程碑（幂等）。

## 注意

- 令牌只用**经典 PAT**（gitcode.com → 设置 → 个人访问令牌(经典)）；新版 JWT 令牌需先换取 access token，脚本不做该转换；
- 限流 400 次/分钟：脚本串行调用够用，勿循环狂刷；
- API 细节以 https://docs.gitcode.com/docs/apis/ 为准，端点调整时只改本脚本。
