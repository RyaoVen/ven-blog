# Changelog

本仓库版本采用语义化版本（SemVer）。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)。

## [v1.2.5] - 2026-08-03

> 注：v1.2.2 之后未单独发布 v1.2.3/v1.2.4，全部变更并入本版。

### 安全加固（安全评审批次）

- **存储型 XSS**：markdown admonition 自定义标题未转义（评论/动态/文章可注入）——补 escapeHtml + 渲染单测
- **内部通道**：VEN_INTERNAL_TOKEN 强制配置（空/默认值 development-token 拒绝启动）、回调鉴权去 fail-open、渲染回调校验 route 归属（防缓存投毒）
- **网关高可用**：Node pattern 拉取失败不再 fatal（磁盘持久化回退启动）、Node 熔断（连续失败快速失败 503 + 半开探测恢复）、页面缓存 stale-while-revalidate（Node 抖动期发过期缓存+异步刷新）
- **认证加固**：登录失败限速锁定（用户名+IP 15 分钟 5 次）、发码限速（邮箱 1 次/分 + IP 每日上限）、种子 author 密码强制配置（未配拒绝启动）、reader 密码 env 可配
- **SSRF**：linkpreview 连接前 DNS 校验拦截私网/回环/云元数据 + 重定向逐跳校验 + 端口白名单
- **会话与跳转**：鉴权 cookie 加 Secure（VEN_COOKIE_SECURE 可配）、登录 next 参数 Open Redirect 修复（站内路径白名单）
- **健壮性**：fiber.Recover 中间件（handler panic → 统一 500）+ 事件总线/SSE goroutine panic 兜底、点赞/收藏 toggle 写优先幂等（消除并发竞态）

### 部署与工程

- deploy config 向导：VEN_INTERNAL_TOKEN 回车自动生成强随机值（或手动输入）、author 密码必填
- deploy 工具 token 必填校验（对齐网关强制语义）
- CI flake 修复：auth 测试请求显式 5s 超时（-race 下 fiber Test 1s 超时）

## [v1.2.2] - 2026-08-03

### 新增

- **订阅邮件通知**：新文章发布异步通知全部订阅者（goroutine 不阻塞发布、单条失败不阻断、panic 兜底）；复用 HTML 模板（标题/摘要/查看原文按钮，站点地址按设置解析）

## [v1.2.1] - 2026-08-03

### 新增

- **站点公网地址设置**：设置面板可配站点公网域名/IP（`site_url`），邮件/RSS 链接站点信息按设置优先、env 兜底；`/api/site` 下发
- **邮件 HTML 模板化**：验证码/@提及/审核摘要三类邮件全面板化——工业极简风格（等宽大号验证码、强调色、细边框、页脚站点信息），html/template 自动转义防注入

### 修复

- register.go 合并残留 `siteURLFromEnv` 未定义（#155/#158 语义冲突，git 自动合并未报但编译失败）

## [v1.2.0] - 2026-08-03

### 新增（国内备案合规）

- **评论总开关**：一键关闭全站评论区（发表 403 + 数据空化 + 前端隐藏；后台审核管理保留）
- **注册登录入口开关**：关闭公开注册与登录入口（注册/邮箱验证码 403 + 导航/登录页隐藏；**作者后台登录通道保留**）

## [v1.1.1] - 2026-08-02

### 修复

- HTML 页面响应无 Cache-Control：手机浏览器启发式缓存部署前旧页面（v1.1.0 移动端文章宽度问题的根因），SSR 页面响应统一 `Cache-Control: no-cache`

## [v1.1.0] - 2026-08-02

### 新增

- **访问统计**：visits 表（按天×路径聚合）+ 双埋点（Go 最外层中间件计整页 PV、SPA 路由上报），仪表盘新增访问量/文章点击总量卡片与近 30 天 PV 折线；文章管理列表每篇显示被点击数/获赞数/收藏数
- **文章与动态置顶**：管理端一键置顶/取消，前台文章列表、动态时间线、首页双列表置顶优先排序并显示"置顶"标记
- **移动端适配**（全站）：顶部导航汉堡抽屉（≤720px）、首页 hero 纵向自适应、正文表格横滚、动态卡/管理列表折行卡片化、后台图表小屏堆叠与横滚、全站按钮触控 ≥40px、移动端面板触摸滚动修复

### 修复

- 动态页头像/作者信息不随资料更新——改资料/改用户名失效声明补 `/moments`（ISR 再生），含 MCP `author.update` 同款遗漏
- CI 盲区：Node job 补 `npm run build`（esbuild 打包是 src/ TSX 的唯一语法守卫）
- CI Go 下载偶发 403：GOPROXY 多级 fallback
- page.tsx 合并残留冲突标记导致的 JSX 构建失败

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
