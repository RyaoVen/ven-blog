# Unit 5：Agent 侧 Skill 工作流设计（publish-post / review-comments / edit-author-page）

- 状态：设计稿（实现阶段落盘 3 个 SKILL.md）
- 依赖：Unit 2 网关 `POST /api/mcp`（本文档撰写时 `docs/agent-design/unit-2-mcp-gateway.md` 尚不存在，payload 字段按"对齐现有 admin 接口字段命名"设计，全部契约点已在【开放问题】中标注，unit-2 定稿后需回核）
- 范围：只设计 agent（ZCode CLI）侧 skill 工作流，不涉及 Go 实现

---

## 1. 目标

把用户三个高频需求做成 3 个**可触发、可执行、可验证**的用户级 skill，全部编排 `/api/mcp` 的 action，代理调用即 author 身份（用户是博客唯一作者）：

| Skill | 触发语境 | 编排 action |
|---|---|---|
| `publish-post` | "帮我发篇文章 / 发个动态" | `post.create`、`moment.create` |
| `review-comments` | "看看有没有新评论 / 审核评论 / 复核" | `comment.list_pending`、`comment.approve/reject/recover` |
| `edit-author-page` | "改个人页 / 改简介 / 加个技能 / 改友链" | `author.get`、`author.update` |

每个 skill 的共性骨架：前置检查 → 确认必填 → 调 action → 错误处理（把服务端 message 回给用户）→ 事后验证（list/页面 200）→ 汇报格式。

最终交付（实现阶段落盘，用户级，风格对齐 `C:\Users\25108\.agents\skills\gh-workflow\SKILL.md`）：

```
C:\Users\25108\.agents\skills\publish-post\SKILL.md
C:\Users\25108\.agents\skills\review-comments\SKILL.md
C:\Users\25108\.agents\skills\edit-author-page\SKILL.md
```

## 2. 契约对齐说明（unit-2 未定稿，先对齐现有接口）

`unit-2-mcp-gateway.md` 不存在，本设计按以下既有事实对齐字段命名（均为 `frame/go/build/interfaces/` 下已定稿的 JSON 契约）：

- **ID 一律字符串下发**：`PostView.ID`、`CommentView.ID`、`MomentView.ID` 均为 string（`dto.go`、`interactions.go`、`moments.go`）。
- **文章字段**：`title / category / content / summary / coverUrl / tags`（`postInput`，`apis.go`）；必填 title/category/content（`post.Validate`，`frame/go/build/domain/post/post.go:60`），summary ≤ 200 字符、coverUrl ≤ 512、tags ≤ 8 个且每个 ≤ 24 字符。
- **动态字段**：`content`（`momentInput`，`moments.go`）；去空白非空且 ≤ 1000 字符（`frame/go/build/domain/moment/moment.go:MaxContentLen`）。
- **评论字段**：`id / userId / username / content / replyTo / status / createdAt`（`CommentView`），后台视图另有 `postId / postTitle`（`adminCommentView`，`admin.go`）。**现状 status 只有 `approved | pending`**（`frame/go/build/domain/comment/comment.go:10`），`reject / recover / reason` 是网关新增能力。
- **作者主页内容**：`settingsapp.Content` = `paragraphs / skills / friends / quotes / projects / github`（`frame/go/build/application/settingsapp/service.go:38`；`src/admin/settingsTypes.ts` 同构）。skill 的 `author.update` payload 对齐该结构。注意：现 `PUT /admin/author/content`（`authorAdmin.go`）只收 paragraphs/skills/friends/projects/showcasePosts、**不含 quotes**，MCP 契约建议按全量 Content 设计（含 quotes），见开放问题 Q4。
- **留言板无审核态**：`guestbook.Entry` 没有 status 字段，发表即公开（`guestbook.go`），因此 `comment.list_pending` **不覆盖留言板**——复核队列只有文章评论与动态评论两类宿主。
- **页面路由**（验证用）：文章详情 `/posts/:id`（`src/posts/[id]/page.tsx`）、动态 `/moments`、作者主页 `/author/:name`（`profiles.go`），均为公开页，HTML 验证不需要鉴权。
- **站点地址**：`BLOG_SITE_URL` 环境变量（`register.go:siteURLFromEnv`，默认 `http://127.0.0.1:8080`）。

## 3. 通用设计（三个 skill 共用）

### 3.1 API key：获取、配置与存放

- **获取**：后台面板生成，格式 `ven_xxx`，**明文仅显示一次**。生成后立即复制。
- **存放（推荐）**：ZCode 所在进程的用户环境变量 **`BLOG_API_KEY`**。
  - 理由：
    1. key 是客户端凭据，**不进仓库**（`.env.local` 虽已 gitignore，但那是服务端运行配置，且团队/多机场景下与仓库内配置混放风险高）；
    2. 环境变量由 ZCode 进程注入，Bash 工具子进程天然继承，skill 正文里 `"$BLOG_API_KEY"` 直接可用，无需任何读取代码；
    3. 命名对齐项目既有 `BLOG_` 前缀（`BLOG_MYSQL_DSN`、`BLOG_SITE_URL`、`BLOG_AUTHOR_NAME`），约定一致；
    4. 重生成 key 时只需覆盖环境变量，不改任何文件。
  - 配置方式（Windows）：`setx BLOG_API_KEY "ven_xxx"`（用户级，新开终端生效），或 ZCode 设置里的环境变量配置项。配好后用一个只读调用自检：`curl -s -H "Authorization: Bearer $BLOG_API_KEY" ...`（见 3.2）。
- **兜底**：若用户不便配置环境变量，可接受一次性会话内 `export BLOG_API_KEY=ven_xxx`，但 skill 前置检查会每次提示"key 未持久化"。
- **站点地址**：复用 `BLOG_SITE_URL`（生产域名），缺省 `http://127.0.0.1:8080`（本地开发）。

### 3.2 curl 封装模板（Windows Git Bash）

统一封装（每个 SKILL.md 内嵌同一函数，或建议后续抽到用户级公共脚本）：

```bash
# 基础封装：mcp <action> <payload-json>；payload 可省略（缺省 {}）
mcp() {
  local action="$1" payload="${2:-{}}"
  curl -sS -X POST "${BLOG_SITE_URL:-http://127.0.0.1:8080}/api/mcp" \
    -H "Authorization: Bearer $BLOG_API_KEY" \
    -H "Content-Type: application/json" \
    --data-raw "{\"action\":\"$action\",\"payload\":$payload}"
}
```

**复杂 payload（长正文/中文/含引号）一律写临时文件再发**，避免引号地狱：

```bash
cat > /tmp/mcp-payload.json <<'EOF'
{"action":"post.create","payload":{"title":"聊聊 Fiber 中间件","category":"技术","content":"# 正文第一行\n\n第二行……","summary":"","coverUrl":"","tags":["fiber","go"]}}
EOF
curl -sS -X POST "${BLOG_SITE_URL:-http://127.0.0.1:8080}/api/mcp" \
  -H "Authorization: Bearer $BLOG_API_KEY" \
  -H "Content-Type: application/json" \
  --data-binary @/tmp/mcp-payload.json
```

Git Bash 引号与转义注意点（本机已踩过、写进 skill 正文）：

1. Git Bash 是 POSIX shell：**单引号内一切不展开**，最适合包 JSON；**双引号内 `$` 和反引号会被展开**——变量（`$BLOG_API_KEY`、`$BLOG_SITE_URL`）必须放双引号里、JSON 字面量放单引号里，两者不要混在同一个引号对中。
2. JSON 内嵌双引号：用单引号包整段即可（`--data-raw '{"a":"b"}'`）；**JSON 里含单引号**（少见，中文文本一般不涉及）时改走临时文件方案。
3. **正文含换行/超长**：命令行 JSON 转义极易出错，一律 `--data-binary @/tmp/xxx.json`。
4. 中文：Git Bash 终端默认 UTF-8 直通，无需 `chcp`；响应乱码先怀疑网关/存储编码问题而非 shell。
5. 不要在 PowerShell 里跑这些命令（`curl` 是 `Invoke-WebRequest` 别名）；必须在 Git Bash 会话。
6. 响应是 JSON 文本，**不强制依赖 jq**（Git Bash 未必装），agent 直接读文本即可；装了 jq 可 `| jq .` 美化。

### 3.3 统一错误处理总表

`/api/mcp` 失败响应形如 `{"error":{"code":"...","message":"..."}}`。skill 遇到错误按此表处理：

| 现象 | code（按 unit-2 约定） | skill 处理 |
|---|---|---|
| 401 鉴权失败 | `invalid_key` | key 无效/已吊销。提示用户回后台面板**重新生成 key**（明文仅一次），重新配置 `BLOG_API_KEY` 后重试；**不**反复重发同一请求 |
| 400 参数校验失败 | `validation` / `bad_request` | 把服务端 `message` **原样回给用户改**（如 "title is required"、"content too long"），改完重发 |
| 404 资源不存在 | `not_found` | 资源可能已被删/被 worker 处理。先 GET 查证再汇报，不盲目重试 |
| 409 状态冲突（若网关实现） | `conflict` | 如评论已被他人处理，刷新列表以最新状态展示 |
| 未知 action / body 解析失败 | `unknown_action` / `bad_body` | **skill 与网关版本不匹配**：检查网关版本与 skill 更新，提示用户，不硬编绕过 |
| 500 服务端错误 | `internal` | 先 GET 查证是否已生效，再决定是否重发（见 3.4） |
| 网络不通 / 超时 | （无响应） | 先确认 `BLOG_SITE_URL` 可达（`curl -s -o /dev/null -w "%{http_code}" $BLOG_SITE_URL`），再 GET 查证写操作是否已成功 |

### 3.4 幂等与重试纪律

- **写操作非幂等**：`post.create`、`moment.create` 每次调用都新建一条；`post.update`、`author.update` 是全量覆盖。**任何失败/超时后不得直接重发**，一律先查证：
  - `post.create` 失败 → `post.list` 按标题+时间找是否已创建；已创建则直接进验证与汇报，未创建才重发。
  - `moment.create` 失败 → `moment.list` 按内容/时间查证。
  - `post.update` / `author.update` 失败 → 先 `post.list`/`author.get` 对比目标字段是否已应用。
- **天然幂等的动作可安全重试**：`comment.approve/reject/recover` 是"置目标状态"，重试无害；但重试前仍先 `comment.list_pending`/`comment.list` 确认当前状态，避免对已处理的评论反复操作。
- 每次重发前向用户说明"已查证未生效，正在重试"，并**最多重试 1 次**，仍失败则停下汇报。

### 3.5 frontmatter 与正文规范（对齐 gh-workflow 风格）

- frontmatter 只要 `name` + `description` 两键；`description` 写**触发语境**（含中文触发词），让 skill 自动触发可靠。
- 正文命令式中文步骤，编号分节；工具类 skill **不需要"授权与红线"节**（无破坏性操作，操作对象都是用户自己的博客）。
- 每步给可直接执行的 Bash 代码块（引用 `mcp` 封装函数与临时文件模板）。
- 汇报格式统一成"✅ + 关键字段"，带 ID 与链接。

## 4. Skill 一：publish-post

### 4.1 触发语境与定位

用户说"帮我发篇文章 / 发个动态 / 发个随想 / 发个碎碎念 / 把这段发到博客"时触发。一篇内容先判断形态：

- **文章（post）**：有标题、有分类、正文成篇 → `post.create`
- **动态（moment）**：短句碎碎念（≤ 1000 字符）、无标题无分类 → `moment.create`

判断不了（比如一句话但用户想当文章发）→ 问用户，不问默认按内容形态判断。

### 4.2 步骤

1. **前置检查**：`BLOG_API_KEY` 已配置（未配置：提示去后台面板生成 key 并配置，中止）；`BLOG_SITE_URL` 缺省 `http://127.0.0.1:8080`。
2. **确认必填（文章）**：标题、分类、正文三项缺一不可，缺的向用户问清楚：
   - 分类缺省提示后台可维护的分类（内置默认：`随笔 / 技术 / 生活`，见 `frame/go/build/application/settingsapp/defaults.go`）；首批 action 没有 categories 查询，分类取值以用户确认的为准（开放问题 Q3）。
   - **正文超长分段建议**：正文约 > 5000 字时，主动建议"拆成系列文章（如《X（一）》/《X（二）》）或先发精简版"，等用户拍板再发；动态正文 > 1000 字符会被网关拒（`moment.MaxContentLen`），超长一律建议改发文章或截断。
   - 可选字段：summary（≤ 200 字符）、coverUrl（≤ 512）、tags（≤ 8 个，各 ≤ 24 字符），有就给，没有可空。
3. **调 action**：

```bash
mcp post.create '{"title":"<标题>","category":"<分类>","content":"<正文>","summary":"","coverUrl":"","tags":[]}'
```

   正文长时走临时文件模板（3.2）。动态：`mcp moment.create '{"content":"<内容>"}'`。
4. **错误处理**：按 3.3 总表；400 时把服务端 `message` 回给用户改（如 "title is required"）。
5. **验证（必做）**：
   - 主验证：`mcp post.list '{}'`（或 `mcp post.list '{"limit":10}'`，limit 字段以 unit-2 契约为准）确认新文章出现在列表（按 createdAt 倒序，查标题匹配）；动态用 `mcp moment.list '{}'`。
   - 冒烟验证：公开页 HTML 200 —— `curl -s -o /dev/null -w "%{http_code}" "${BLOG_SITE_URL:-http://127.0.0.1:8080}/posts/<id>"`（文章）/ `.../moments`（动态）。文章详情页是 ISR 共享物化，发布后失效再生有异步窗口，**页面 200 即通过**，内容一致性以 list 为准。
6. **汇报格式**：

```text
✅ 文章已发布
- ID: 42
- 标题: 聊聊 Fiber 中间件
- 分类: 技术
- 摘要: 从中间件注册顺序讲起……（未填摘要时：已自动截取正文前 80 字）
- 标签: fiber / go
- 链接: http://127.0.0.1:8080/posts/42
```

   动态汇报：`✅ 动态已发布 · ID 7 · 链接 http://127.0.0.1:8080/moments`（内容过长只回显前 100 字）。

### 4.3 frontmatter 与正文草稿

````markdown
---
name: publish-post
description: 发布博客文章或动态：用户说"帮我发篇文章/发个动态/发个随想/发个碎碎念/把这段发到博客"时使用。走 POST /api/mcp 的 post.create / moment.create，确认必填字段、正文超长给分段建议、发布后经 post.list/moment.list 与页面 200 验证并汇报（含 ID、URL、分类、摘要）。
---

# 发布文章 / 动态

## 0. 前置检查

- `test -n "$BLOG_API_KEY"`：未配置则提示"去博客后台面板生成 API key（ven_xxx，明文仅显示一次），配置到环境变量 BLOG_API_KEY 后重试"，然后停止。
- 站点地址：`BLOG_SITE_URL`（缺省 http://127.0.0.1:8080）。
- curl 封装见下（mcp 函数）；payload 含换行/超长正文时写临时文件再发。

```bash
mcp() {
  local action="$1" payload="${2:-{}}"
  curl -sS -X POST "${BLOG_SITE_URL:-http://127.0.0.1:8080}/api/mcp" \
    -H "Authorization: Bearer $BLOG_API_KEY" \
    -H "Content-Type: application/json" \
    --data-raw "{\"action\":\"$action\",\"payload\":$payload}"
}
```

## 1. 判断形态：文章 or 动态

- 有标题/分类、正文成篇 → 文章（步骤 2）
- 短句碎碎念、无标题 → 动态（跳到步骤 4）
- 拿不准 → 问用户

## 2. 文章：确认必填

- 标题、分类、正文缺一不可，缺的向用户问清楚。分类默认可选：随笔 / 技术 / 生活。
- 正文约超过 5000 字：建议拆系列（《X（一）》…）或先发精简版，等用户拍板。
- 可选：summary（≤200 字）、coverUrl、tags（≤8 个，各 ≤24 字符）。

## 3. 文章：post.create → 验证 → 汇报

- 短 payload 用 mcp 函数；长正文写 /tmp/mcp-payload.json 后 `--data-binary @`。
- 400 时把服务端 message 原样回给用户改。
- 验证：`mcp post.list '{}'` 确认新文章在列表头；`curl -s -o /dev/null -w "%{http_code}" "$BLOG_SITE_URL/posts/<id>"` 期望 200。
- 汇报：✅ 文章已发布 · ID / 标题 / 分类 / 摘要 / 链接。

## 4. 动态：moment.create → 验证 → 汇报

- `mcp moment.create '{"content":"..."}'`；内容超 1000 字符会被拒，超长建议改发文章或截断。
- 验证：`mcp moment.list '{}'` 确认在列；`curl -s -o /dev/null -w "%{http_code}" "$BLOG_SITE_URL/moments"` 期望 200。
- 汇报：✅ 动态已发布 · ID / 链接 / 内容前 100 字。

## 5. 失败与重试

- 401 invalid_key：提示重新生成 key，不重发。
- 5xx/超时：先 list 查证是否已创建，未创建才重发，最多 1 次。
````

## 5. Skill 二：review-comments

### 5.1 触发语境与定位

用户说"看看有没有新评论 / 审核评论 / 复核 / 评论审核 / 评论去哪了"时触发。

定位：**worker 自动审核的兜底通道**。正常流程里评论由 worker 自动处理（自动放行/拒绝；拿不准的"不确定态"评论会邮件通知作者），作者在邮件里看到后可以来本 skill 复核——因此本 skill 的展示必须包含 worker 留下的判定信息（rejected 的 `reason` 字段）。

**宿主范围**：只覆盖评论（宿主为文章或动态）。留言板（guestbook）无审核态、发表即公开，不在队列里（契约说明见第 2 节）。

### 5.2 步骤

1. **前置检查**：同 4.2 步骤 1。
2. **拉队列**：

```bash
mcp comment.list_pending '{}'
```

   期望 `data.comments` 为数组。若 unit-2 契约在响应里同时给了近期已处理记录（如 `autoProcessed`/`recent` 字段），一并展示（见 3 的展示格式）；契约没给就只处理 pending。
3. **逐条展示**（每条：内容 / 宿主 / 评论者 / 时间 / 状态）：

```text
#12  [文章《聊聊 Fiber 中间件》]  @小明  2026-08-02 10:30  pending
    "写得好，学到了！"
#9   [动态 #7]  @路人甲  2026-08-02 09:12  rejected
    原因：疑似广告（含外链）      ← rejected 必须展示 reason
```

   - 宿主信息：文章显示 postTitle（无则 postId），动态显示 momentId；`replyTo` 非空时标注"回复 @xx"。
   - 队列为空 → 汇报"✅ 没有待审核评论"，并提示"worker 已自动处理的部分可在后台评论页查看；不确定态会邮件通知你"。
4. **用户拍板**：对每条给建议动作（approve 放行 / reject 拒绝 / recover 恢复），等用户确认后执行：

```bash
mcp comment.approve '{"id":"12"}'
mcp comment.reject '{"id":"9","reason":"广告"}'   # reject 建议带 reason，网关契约允许则必填/可选以 unit-2 为准
mcp comment.recover '{"id":"9"}'                   # 把误 reject 的评论恢复（恢复为 pending 或 approved 以 unit-2 契约为准）
```

   用户也可以一句话批量拍板（"3、5 通过，6 拒了"），逐条执行并各自汇报。
5. **每动作后汇报**：

```text
✅ 已通过评论 #12（文章《聊聊 Fiber 中间件》）
✅ 已拒绝评论 #9（动态 #7），原因：广告
```

   404/409 → 提示"该评论可能已被 worker 或他处处理，已刷新列表"，重新拉队列确认。
6. **收尾**：全部处理完汇报一次汇总：`共 3 条：通过 2 / 拒绝 1`；列表还有剩余时说明剩余条数，问是否继续。

### 5.3 frontmatter 与正文草稿

````markdown
---
name: review-comments
description: 复核评论审核队列（worker 自动审核的兜底通道）：用户说"看看有没有新评论/审核评论/复核/评论去哪了"时使用。拉 comment.list_pending 逐条展示待审评论（内容/宿主/评论者/时间/状态，rejected 展示 worker 填的 reason），按用户拍板 approve/reject/recover，每动作后汇报。
---

# 复核评论审核队列

## 0. 前置检查

- `test -n "$BLOG_API_KEY"`：未配置则提示配置方法（同 publish-post），停止。
- mcp 封装函数与临时文件模板同 publish-post。

## 1. 拉取待审核队列

```bash
mcp comment.list_pending '{}'
```

- 响应里若有近期已处理记录一并展示（rejected 必须带 reason）。
- 队列为空：汇报"没有待审核评论"，并提示 worker 自动处理、不确定态会邮件通知作者。

## 2. 逐条展示

每条格式：#ID [宿主] @评论者 时间 状态 + 内容；rejected 必须展示 reason；replyTo 非空标注"回复 @xx"。
宿主：文章显示 postTitle（缺省 postId），动态显示 momentId。留言板无审核态，不在此列。

## 3. 用户拍板并执行

- 默认建议：正常讨论 approve；广告/垃圾 reject（附 reason）；误伤 recover。
- 逐条执行：

```bash
mcp comment.approve '{"id":"12"}'
mcp comment.reject '{"id":"9","reason":"广告"}'
mcp comment.recover '{"id":"9"}'
```

## 4. 汇报

- 每个动作后：✅ 已通过/已拒绝/已恢复 评论 #ID（宿主）。
- 404/409：提示该评论已被处理，重新拉队列。
- 全部处理完：汇总条数（通过 n / 拒绝 n），有剩余则说明并询问是否继续。
````

## 6. Skill 三：edit-author-page

### 6.1 触发语境与定位

用户说"改个人页 / 改简介 / 加个技能 / 改友链 / 改项目 / 改签名 / 换引用"时触发。编辑对象是作者主页 `/author/:name` 的内容配置（`settingsapp.Content` 结构：`paragraphs / skills / friends / quotes / projects / github`，见 `frame/go/build/application/settingsapp/service.go:38`）。

**全量覆盖语义**：`author.update` 一次提交完整 Content（对齐后台设置页保存行为），skill 必须"先 get 再改再全量提交"，未提及的字段保持原值，防止把没让改的部分清空。

### 6.2 步骤

1. **前置检查**：同 4.2 步骤 1。
2. **拉当前内容**：

```bash
mcp author.get '{}'
```

   期望 `data` 含 `username`（作者名，拼页面 URL 用）与 `content`（或平铺 content 字段，以 unit-2 契约为准）。
3. **按用户指令修改**：agent 在 get 结果上改对应字段：
   - 简介 → `paragraphs`（段落数组，增/删/改某段）
   - 技能 → `skills`（`[{name, level}]`，level 枚举 deep/solid/know，见 `defaults.go`）
   - 友链 → `friends`（`[{name, url, desc}]`）
   - 签名/引用 → `quotes`（`[{text, source}]`）
   - 项目 → `projects`（`[{name, desc, url}]`）
   - 结构性改动（增删技能/友链）先给用户看"改前 → 改后"再提交。
4. **调 action（全量提交）**：

```bash
mcp author.update '{"content":{"paragraphs":[...],"skills":[{"name":"Go","level":"deep"}],"friends":[...],"quotes":[...],"projects":[...],"github":"https://github.com/RyaoVen"}}'
```

   payload 结构与 unit-2 契约对齐（平铺 content 字段或嵌套 content 对象以 unit-2 为准，正文草稿按嵌套写、实现时对齐）。
5. **验证**：
   - 主验证：`mcp author.get '{}'` 重新拉取，**对比改动字段逐项确认已生效**。
   - 冒烟验证：`curl -s -o /dev/null -w "%{http_code}" "${BLOG_SITE_URL:-http://127.0.0.1:8080}/author/<username>"` 期望 200（作者页是动态页，更新后网关会失效声明；200 只证明路由在，内容一致性以 get 对比为准）。
6. **汇报格式**：

```text
✅ 个人主页已更新
- 改了什么: 简介加了一段"喜欢写博客"；新增技能 TypeScript(solid)
- 未改动: 友链 / 项目 / 引用
- 页面: http://127.0.0.1:8080/author/author
```

### 6.3 frontmatter 与正文草稿

````markdown
---
name: edit-author-page
description: 修改个人主页内容：用户说"改个人页/改简介/加个技能/改友链/改项目/改签名/换引用"时使用。author.get 拉当前内容（paragraphs/skills/friends/quotes/projects 结构）→ 按指令修改 → author.update 全量提交（未提及字段保持原值）→ 重新 get 对比验证 + 作者页 200 冒烟 → 汇报改动。
---

# 修改个人主页

## 0. 前置检查

- `test -n "$BLOG_API_KEY"`：未配置则提示配置方法，停止。
- mcp 封装函数与临时文件模板同 publish-post。

## 1. 拉取当前内容

```bash
mcp author.get '{}'
```

- 记录 username（拼页面 URL）与全部 content 字段。

## 2. 按指令修改

- 简介 → paragraphs（数组）；技能 → skills[{name,level}]（level: deep/solid/know）；友链 → friends[{name,url,desc}]；引用 → quotes[{text,source}]；项目 → projects[{name,desc,url}]。
- 结构性增删先给用户看"改前 → 改后"。
- 没让改的字段一律保持 get 的原值（update 是全量覆盖，漏传等于清空）。

## 3. 全量提交

```bash
mcp author.update '{"content":{...完整 Content...}}'
```

- 400 时把服务端 message 回给用户改。

## 4. 验证

- `mcp author.get '{}'` 重新拉取，逐项对比改动字段已生效。
- `curl -s -o /dev/null -w "%{http_code}" "$BLOG_SITE_URL/author/<username>"` 期望 200。

## 5. 汇报

- ✅ 个人主页已更新：改了什么（逐条）/ 未改动 / 页面链接。
````

## 7. 验收标准（人工可验收）

| # | 触发句 | 预期行为 |
|---|---|---|
| P1 | "帮我发篇文章：《聊聊 Fiber 中间件》，分类技术，正文……" | 自动 post.create；汇报含 ID、URL、分类、摘要；`/posts/<id>` 打开 200 且内容一致 |
| P2 | "帮我发篇文章，正文是……"（缺标题/分类） | 反问补标题/分类后才发，不擅自造 |
| P3 | 正文 8000 字时发文章 | 先给"拆系列/精简版"建议，用户确认后按确认结果执行 |
| P4 | "发个动态：今天 CI 通了" | moment.create；汇报含 ID、链接；/moments 可见 |
| P5 | 未配置 BLOG_API_KEY 时触发任一发布 | 明确提示去面板生成 key 并配置环境变量，不发起请求 |
| R1 | "看看有没有新评论"（有 pending） | 逐条展示 内容/宿主/评论者/时间/状态 |
| R2 | "把 #3 拒了" | comment.reject 并汇报；rejected 条目标注 reason |
| R3 | "把 #5 恢复" | comment.recover 并汇报；误 reject 后状态恢复 |
| R4 | 队列为空 | 汇报"没有待审核评论"+ worker 自动处理/邮件通知提示 |
| R5 | 对已处理评论重复 approve | 404/409 提示已被处理并刷新列表，不报错不重试 |
| E1 | "改简介，加一句'喜欢写博客'" | author.get → update → 重新 get 对比该段生效；/author/<name> 200 |
| E2 | "加个技能：Rust(know)" | skills 数组追加且其余字段原样保留（全量覆盖不丢数据） |
| E3 | "改友链：把第一个换成 xxx" | friends 第一项替换成功，其余不动 |
| E4 | 只改简介时故意漏传友链（模拟） | 按 skill 步骤必须先 get 全量再提交，友链不被清空（验证全量覆盖纪律） |

## 8. 开放问题（unit-2 定稿前不回合）

1. **Q1 评论状态机**：现领域只有 `approved | pending`；`reject / recover / reason` 为网关新增。需 unit-2 确认：status 枚举（是否加 `rejected`）、`reason` 字段落库、`recover` 的目标态（回 pending 还是直接 approved）、rejected 是否进入 `comment.list`。
2. **Q2 留言板**：guestbook 无审核态。本设计复核队列不含留言板；若 unit-2 给留言板加审核，review-comments 需扩展宿主维度。
3. **Q3 分类**：首批 action 无 categories 查询，publish-post 只能靠用户口述分类。建议 unit-2 增加 `category.list`（或 post.list 响应带可用分类），否则发错分类只能事后 update 修正。
4. **Q4 author.get/update 契约**：payload 是否含 `quotes` 与 `github`（现 `PUT /admin/author/content` 不收 quotes，而设置页/`settingsapp.Content` 有）；`showcasePosts`（展示柜文章 ID 数组）是否纳入 MCP 范围；author.get 是否返回 username（拼页面 URL 用）。
5. **Q5 正文长度上限**：post 领域无硬限制（仅动态 1000 字符）；本设计"5000 字分段建议"是体验阈值，若网关加了硬上限，publish-post 的超长分支以服务端 message 为准即可，无需改 skill。
6. **Q6 worker 语境**：本设计引用的"评论自动审核 worker、不确定态邮件通知、reason 字段"来自需求描述，代码中尚未落地（AGENTS.md 中的 worker 指 Node SSR 渲染 worker）。review-comments 的 worker 相关展示逻辑以 worker 单元落地后的实际契约为准。
7. **Q7 list 响应结构**：`post.list / moment.list / comment.list / comment.list_pending` 的响应字段（分页参数如 limit、是否含已处理记录）未定稿，skill 草稿按"数组 + 顶层字段"编写，实现时对齐 unit-2。
8. **Q8 展示柜**：作者页"展示柜"（showcasePosts）涉及文章 ID 选择，属高频还是低频未定；本设计未纳入 edit-author-page（可通过 author.get/update 扩展），如用户需要再补。

## 9. 实现阶段清单（供后续单元）

- [ ] unit-2 网关 `/api/mcp` 落地并回核本设计中的契约假设（Q1–Q8）
- [ ] 落盘 3 个 SKILL.md 到 `C:\Users\25108\.agents\skills\{publish-post,review-comments,edit-author-page}\SKILL.md`
- [ ] 用户侧配置 `BLOG_API_KEY`（面板生成 → setx / ZCode 环境变量）
- [ ] 按第 7 节验收表逐条人工验收（本地起站 8080）
- [ ] 视验收结果决定是否把 `mcp()` 封装抽到用户级公共脚本（避免三份拷贝）
