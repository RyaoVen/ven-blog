# Changelog

本仓库版本采用语义化版本（SemVer）。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)。

## [v1.0.0] - 2026-08-02

首个正式版本：VenHybird 混合渲染框架 + ven-blog 博客业务 + AI agent 能力 + 跨平台部署工具，CI 全绿封仓。

### 框架（VenHybird）

- Go(Fiber) 网关 + Node SSR worker 混合渲染：首屏 SSR 直出、SPA 接管、ISR 静态页物化直发
- 事件总线：数据变更 debounce 合批（5s 静默窗口/30s 强制）、先删后渲、页面级代际防覆盖、map 去重
- 页面缓存（1min TTL、防击穿）+ 鉴权守卫（角色继承、会话 cookie、登录重定向）
- 集群：Redis 会话/缓存后端即跨实例共享（docs/cluster.md）

### 博客业务（ven-blog）

- 文章（分类/标签/封面/多内容块）、动态、评论（先审后发开关）、留言板、点赞/收藏、订阅（RSS 邮件）、搜索、站点设置
- 后台管理面板：仪表盘图表、文章/评论/动态/留言板管理、个人主页分项编辑（介绍/技能/友链/项目/展示柜）、分类管理、站点配置（图标/SMTP/审核开关/LLM）

### AI Agent 能力

- **API 密钥**：面板自助生成/吊销（明文仅显示一次、服务端只存 sha256、多 key 分用途、吊销即时生效）
- **/api/mcp 网关**：Bearer key 鉴权、JSON-RPC 分发（文章/动态/评论审核/个人主页 14 个 action）、统一错误码
- **自动审核 worker**：LLM（OpenAI 兼容）判定评论+留言板，驳回（记原因）/放行/不确定挂起三态；失败安全（审不了不误杀）；摘要邮件去重节流；面板开关与 LLM 配置
- **ZCode skill**：publish-post / review-comments / edit-author-page（~/.agents/skills/，配合 BLOG_API_KEY）

### 部署与工程

- **tools/deploy**：跨平台（Windows/Linux）部署工具——bubbletea TUI + 子命令（check/config/build/start/stop/restart/status/logs）；端口可配（VEN_NODE_PORT/VEN_LISTEN_ADDR）；Go 就绪等待（/api/site 15s）；进程 detach（关终端不杀）；环境检测与配置向导
- **CI**：GitHub Actions 三件套（Go -race / Node typecheck+test / 部署工具），Node 跨平台 optional 依赖用 npm install 补全
- 使用文档 docs/usage.md、设计文档 docs/agent-design/、工具文档 tools/deploy/README.md

### 修复

- 评论/留言板三态审核语义与公开查询只返 approved 的行为变更
- 面板 reject 端点宿主失效（approved→rejected 可见性变化）
- 部署工具评审四项：测试隔离（preflight 顺序）、Go 就绪校验、端口可配、Windows detach

[v1.0.0]: https://github.com/RyaoVen/ven-blog/releases/tag/v1.0.0
