// /api/mcp 网关（agent 统一入口）：外部 AI/脚本持 API key 经单一端点执行博客管理操作。
// 协议与错误码见 docs/agent-design/unit-2-mcp-gateway.md。
// 关键约束：只认 Authorization: Bearer ven_xxx，不认 cookie，不参与 hybrid cookie 鉴权链，
// 401 不设置 X-Ven-Login-Path；写操作后的失效声明复用同包辅助函数。
package interfaces

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"ven_hybird/build/application/commentapp"
	"ven_hybird/build/application/momentapp"
	"ven_hybird/build/application/postapp"
	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/application/userapp"
	"ven_hybird/build/domain/comment"
	"ven_hybird/build/domain/moment"
	"ven_hybird/build/domain/post"
	"ven_hybird/build/domain/user"
	"ven_hybird/hybrid"
)

/* ===== 协议与错误码（§3） ===== */

// 协议错误码（与 HTTP 状态一一对应；forbidden 预留——首批 action 均为 author 自有操作）。
const (
	mcpCodeInvalidKey = "invalid_key" // 401：缺 key / 格式错 / AuthenticateKey 失败（含吊销）
	mcpCodeBadRequest = "bad_request" // 400：非法 JSON、缺 action、未知 action、payload 非对象；413 的码
	mcpCodeValidation = "validation"  // 400：应用层 *ValidationError 或契约级字段校验失败
	mcpCodeNotFound   = "not_found"   // 404：post/moment/comment/user 领域 ErrNotFound
	mcpCodeForbidden  = "forbidden"   // 403：预留（Unit 3+ 权限分支）
	mcpCodeInternal   = "internal"    // 500：其余未分类错误
)

// maxMCPBodyBytes 请求体大小预检上限（1MB）。
// 全局 fiber BodyLimit 为 10MB（httpserver/server.go），但全局 ErrorHandler 会把
// fiber 触发的 413 统一写成 500，故中间件显式按 Content-Length 预检，超限直接 413。
const maxMCPBodyBytes = 1024 * 1024

// mcpRequest 是 /api/mcp 请求体。
// 命名 JSON-RPC 风格，但不是 JSON-RPC 2.0（无 jsonrpc/id 字段——单端点无并发关联语义）。
type mcpRequest struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"` // 缺省归一为 {}
}

// mcpError 是协议错误：HTTP 状态码 + 协议错误码 + 消息（dispatcher 统一写响应体）。
type mcpError struct {
	HTTPStatus int
	Code       string
	Message    string
}

// mcpErr 构造协议错误。
func mcpErr(status int, code, message string) *mcpError {
	return &mcpError{HTTPStatus: status, Code: code, Message: message}
}

// writeMCPError 写协议错误响应体。
// 401 不设置 X-Ven-Login-Path——该头仅 web cookie 鉴权链使用（hybrid/api.go），
// agent 场景无效且会误导客户端去重定向。
func writeMCPError(ctx *fiber.Ctx, merr *mcpError) error {
	return ctx.Status(merr.HTTPStatus).JSON(fiber.Map{
		"error": fiber.Map{"code": merr.Code, "message": merr.Message},
	})
}

/* ===== 鉴权中间件（§4） ===== */

// KeyAuthenticator 是 API key 校验的最小契约（*apikeyapp.Service.AuthenticateKey 满足：
// AuthenticateKey(rawKey string) (userID int64, err error)）。
type KeyAuthenticator interface {
	AuthenticateKey(rawKey string) (int64, error)
}

// mcpCtxKey 类型化 Locals key，避免与 fiber 内建/其他中间件冲突。
type mcpCtxKey string

// mcpUserIDKey 注入的 key 持有者 userID（int64）。
const mcpUserIDKey mcpCtxKey = "mcp.userID"

// mcpUserID 取出中间件注入的 userID（断言失败视为 internal，理论不可达）。
func mcpUserID(ctx *fiber.Ctx) int64 {
	id, _ := ctx.Locals(mcpUserIDKey).(int64)
	return id
}

// mcpAuth 鉴权中间件：请求体大小预检 → Bearer 解析 → 格式预检 → AuthenticateKey → 注入身份。
// 不读 cookie、不调 CookieAuth/CurrentUser——本入口与面板会话零耦合；
// 鉴权失败不区分原因（未知/吊销/格式错/后端错统一 invalid_key，不泄露 key 是否存在）。
func (m *MCP) mcpAuth(ctx *fiber.Ctx) error {
	// 请求体大小预检（Content-Length 缺失为 -1，跳过预检，由全局 10MB 兜底）
	if n := ctx.Request().Header.ContentLength(); n > maxMCPBodyBytes {
		return ctx.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"error": fiber.Map{"code": mcpCodeBadRequest, "message": "payload too large"},
		})
	}

	header := ctx.Get(fiber.HeaderAuthorization)
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return writeMCPError(ctx, mcpErr(fiber.StatusUnauthorized, mcpCodeInvalidKey, "invalid api key"))
	}
	rawKey := strings.TrimSpace(parts[1])
	if !strings.HasPrefix(rawKey, "ven_") {
		// 格式预检快速拒绝（不调 AuthenticateKey）
		return writeMCPError(ctx, mcpErr(fiber.StatusUnauthorized, mcpCodeInvalidKey, "invalid api key"))
	}
	userID, err := m.keys.AuthenticateKey(rawKey)
	if err != nil {
		// 未知/吊销/仓储内部错统一 invalid_key（不泄露 key 是否存在）
		return writeMCPError(ctx, mcpErr(fiber.StatusUnauthorized, mcpCodeInvalidKey, "invalid api key"))
	}
	if userID <= 0 {
		// 防御分支：理论不可达
		return writeMCPError(ctx, mcpErr(fiber.StatusInternalServerError, mcpCodeInternal, "internal error"))
	}
	ctx.Locals(mcpUserIDKey, userID)
	return ctx.Next()
}

/* ===== action 分发（§5） ===== */

// mcpActionFunc action 处理函数签名：payload 已归一为非空对象 JSON；返回 data（成功）或 *mcpError。
type mcpActionFunc func(m *MCP, ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError)

// mcpActions dispatch 表：action 名 → 处理函数（注册期构建，运行期只读，天然并发安全）。
var mcpActions = map[string]mcpActionFunc{
	"post.create":           (*MCP).postCreate,
	"post.update":           (*MCP).postUpdate,
	"post.delete":           (*MCP).postDelete,
	"post.list":             (*MCP).postList,
	"moment.create":         (*MCP).momentCreate,
	"moment.delete":         (*MCP).momentDelete,
	"moment.list":           (*MCP).momentList,
	"comment.list_pending":  (*MCP).commentListPending,
	"comment.approve":       (*MCP).commentApprove,
	"comment.reject":        (*MCP).commentReject,
	"comment.recover":       (*MCP).commentRecover,
	"comment.list":          (*MCP).commentList,
	"author.get":            (*MCP).authorGet,
	"author.update":         (*MCP).authorUpdate,
}

// MCP 是 /api/mcp 网关处理器：组装根注入全部依赖。
type MCP struct {
	a            *hybrid.App
	keys         KeyAuthenticator
	posts        *postapp.Service
	moments      *momentapp.Service
	comments     *commentapp.Service
	settings     *settingsapp.Service
	users        *userapp.Service
	authorFn     func() (*user.User, error)
	authorNameFn func() string
}

// RegisterMCP 注册 /api/mcp 网关（原生 fiber 路由，不走 hybrid cookie 鉴权链）。
// 先例 auth.go：server := a.Server(); server.App().Post(path, ...)。
// /api 前缀对原生 fiber 路由不生效，必须写全路径；禁止再经 a.Post("/mcp", ...) 注册同名路由。
func RegisterMCP(
	a *hybrid.App,
	keys KeyAuthenticator,
	posts *postapp.Service,
	moments *momentapp.Service,
	comments *commentapp.Service,
	settings *settingsapp.Service,
	users *userapp.Service,
	authorFn func() (*user.User, error),
	authorNameFn func() string,
) error {
	m := &MCP{
		a:            a,
		keys:         keys,
		posts:        posts,
		moments:      moments,
		comments:     comments,
		settings:     settings,
		users:        users,
		authorFn:     authorFn,
		authorNameFn: authorNameFn,
	}
	a.Server().App().Post("/api/mcp", m.mcpAuth, m.handle)
	return nil
}

// handle POST /api/mcp 主流程：解析协议 → dispatch → 统一响应。
func (m *MCP) handle(ctx *fiber.Ctx) error {
	var req mcpRequest
	if err := json.Unmarshal(ctx.Body(), &req); err != nil {
		// 非法 JSON / action 非字符串（类型错误）统一 bad_request
		return writeMCPError(ctx, mcpErr(fiber.StatusBadRequest, mcpCodeBadRequest, "invalid request body"))
	}
	if req.Action == "" {
		return writeMCPError(ctx, mcpErr(fiber.StatusBadRequest, mcpCodeBadRequest, "action is required"))
	}
	h, ok := mcpActions[req.Action]
	if !ok {
		return writeMCPError(ctx, mcpErr(fiber.StatusBadRequest, mcpCodeBadRequest, "unknown action: "+req.Action))
	}
	payload, merr := normalizePayload(req.Payload)
	if merr != nil {
		return writeMCPError(ctx, merr)
	}
	data, merr := h(m, ctx, payload)
	if merr != nil {
		return writeMCPError(ctx, merr)
	}
	return ctx.JSON(fiber.Map{"ok": true, "data": data})
}

// normalizePayload 归一 payload：缺省/null → {}；其余非对象 → bad_request。
func normalizePayload(raw json.RawMessage) (json.RawMessage, *mcpError) {
	payload := bytes.TrimSpace(raw)
	if len(payload) == 0 || bytes.Equal(payload, []byte("null")) {
		return json.RawMessage(`{}`), nil
	}
	if payload[0] != '{' {
		return nil, mcpErr(fiber.StatusBadRequest, mcpCodeBadRequest, "payload must be an object")
	}
	return payload, nil
}

// decodePayload 把 action payload 解到结构体；类型不匹配视为 bad_request。
func decodePayload(payload json.RawMessage, v any) *mcpError {
	if err := json.Unmarshal(payload, v); err != nil {
		return mcpErr(fiber.StatusBadRequest, mcpCodeBadRequest, "invalid payload")
	}
	return nil
}

/* ===== 错误映射（复用 writePostError/writeMomentError 判定分支，响应体改协议格式） ===== */

// classifyPostError 文章用例错误 → 协议错误。
func classifyPostError(err error) *mcpError {
	var vErr *postapp.ValidationError
	switch {
	case errors.As(err, &vErr):
		return mcpErr(fiber.StatusBadRequest, mcpCodeValidation, vErr.Message)
	case errors.Is(err, post.ErrNotFound):
		return mcpErr(fiber.StatusNotFound, mcpCodeNotFound, "post not found")
	default:
		return mcpErr(fiber.StatusInternalServerError, mcpCodeInternal, "internal error")
	}
}

// classifyMomentError 动态用例错误 → 协议错误。
func classifyMomentError(err error) *mcpError {
	var vErr *momentapp.ValidationError
	switch {
	case errors.As(err, &vErr):
		return mcpErr(fiber.StatusBadRequest, mcpCodeValidation, vErr.Message)
	case errors.Is(err, moment.ErrNotFound):
		return mcpErr(fiber.StatusNotFound, mcpCodeNotFound, "moment not found")
	default:
		return mcpErr(fiber.StatusInternalServerError, mcpCodeInternal, "internal error")
	}
}

// classifyCommentError 评论用例错误 → 协议错误。
func classifyCommentError(err error) *mcpError {
	switch {
	case errors.Is(err, comment.ErrNotFound):
		return mcpErr(fiber.StatusNotFound, mcpCodeNotFound, "comment not found")
	case errors.Is(err, comment.ErrInvalidState):
		return mcpErr(fiber.StatusBadRequest, mcpCodeValidation, "comment not in rejected state")
	}
	var vErr *commentapp.ValidationError
	if errors.As(err, &vErr) {
		return mcpErr(fiber.StatusBadRequest, mcpCodeValidation, vErr.Message)
	}
	return mcpErr(fiber.StatusInternalServerError, mcpCodeInternal, "internal error")
}

// classifyUserError 用户用例错误 → 协议错误（ErrUsernameTaken 无 conflict 码，归入 validation）。
func classifyUserError(err error) *mcpError {
	switch {
	case errors.Is(err, user.ErrNotFound):
		return mcpErr(fiber.StatusNotFound, mcpCodeNotFound, "user not found")
	case errors.Is(err, user.ErrUsernameTaken):
		return mcpErr(fiber.StatusBadRequest, mcpCodeValidation, "username taken")
	}
	var vErr *userapp.ValidationError
	if errors.As(err, &vErr) {
		return mcpErr(fiber.StatusBadRequest, mcpCodeValidation, vErr.Message)
	}
	return mcpErr(fiber.StatusInternalServerError, mcpCodeInternal, "internal error")
}

/* ===== action handlers（§6） ===== */

// postCreate 发布文章（6.1）：归属 key 校验出的 userID。
func (m *MCP) postCreate(ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError) {
	var in postInput
	if merr := decodePayload(payload, &in); merr != nil {
		return nil, merr
	}
	p, err := m.posts.Create(mcpUserID(ctx), in.toServiceInput())
	if err != nil {
		return nil, classifyPostError(err)
	}
	declarePostsChanged(m.a, p.ID)
	// 作者用户页文章数 +1（与 web 发文接口对齐）
	m.a.InvalidatePage("/users/" + m.authorNameFn())
	return fiber.Map{"id": strconv.FormatInt(p.ID, 10)}, nil
}

// postUpdate 编辑文章（6.2）。
func (m *MCP) postUpdate(ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError) {
	var in struct {
		ID string `json:"id"`
		postInput
	}
	if merr := decodePayload(payload, &in); merr != nil {
		return nil, merr
	}
	p, err := m.posts.Update(mustID(in.ID), in.postInput.toServiceInput())
	if err != nil {
		return nil, classifyPostError(err)
	}
	declarePostsChanged(m.a, p.ID)
	return fiber.Map{"post": toPostView(p)}, nil
}

// postDelete 删除文章（6.3）。
func (m *MCP) postDelete(ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError) {
	var in struct {
		ID string `json:"id"`
	}
	if merr := decodePayload(payload, &in); merr != nil {
		return nil, merr
	}
	id := mustID(in.ID)
	if err := m.posts.Delete(id); err != nil {
		return nil, classifyPostError(err)
	}
	declarePostsChanged(m.a, id)
	// 作者用户页文章数 -1
	m.a.InvalidatePage("/users/" + m.authorNameFn())
	return fiber.Map{"deleted": true}, nil
}

// postList 文章列表（6.4）：只给 limit 走 ListRecent（limit<=0 全部）；否则走分页 List。
func (m *MCP) postList(ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError) {
	var in struct {
		Limit    *int   `json:"limit"`
		Category string `json:"category"`
		Page     int    `json:"page"`
		PageSize int    `json:"pageSize"`
	}
	if merr := decodePayload(payload, &in); merr != nil {
		return nil, merr
	}
	if in.Limit != nil {
		list, err := m.posts.ListRecent(*in.Limit)
		if err != nil {
			return nil, mcpErr(fiber.StatusInternalServerError, mcpCodeInternal, "internal error")
		}
		return fiber.Map{"posts": toPostViews(list)}, nil
	}
	paged, err := m.posts.List(postapp.ListFilter{Category: in.Category, Page: in.Page, PageSize: in.PageSize})
	if err != nil {
		return nil, mcpErr(fiber.StatusInternalServerError, mcpCodeInternal, "internal error")
	}
	return fiber.Map{
		"posts":    toPostViews(paged.Posts),
		"total":    paged.Total,
		"page":     paged.Page,
		"pageSize": paged.PageSize,
	}, nil
}

// momentCreate 发布动态（6.5）：归属 key 校验出的 userID。
func (m *MCP) momentCreate(ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError) {
	var in momentInput
	if merr := decodePayload(payload, &in); merr != nil {
		return nil, merr
	}
	mm, err := m.moments.Create(mcpUserID(ctx), in.Content)
	if err != nil {
		return nil, classifyMomentError(err)
	}
	_ = m.a.DataChange("/moments")
	return fiber.Map{"id": strconv.FormatInt(mm.ID, 10)}, nil
}

// momentDelete 删除动态（6.6）。
func (m *MCP) momentDelete(ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError) {
	var in struct {
		ID string `json:"id"`
	}
	if merr := decodePayload(payload, &in); merr != nil {
		return nil, merr
	}
	if err := m.moments.Delete(mustID(in.ID)); err != nil {
		return nil, classifyMomentError(err)
	}
	_ = m.a.DataChange("/moments")
	return fiber.Map{"deleted": true}, nil
}

// momentList 动态列表（6.7）：最多 50 条。
func (m *MCP) momentList(ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError) {
	list, err := m.moments.List()
	if err != nil {
		return nil, mcpErr(fiber.StatusInternalServerError, mcpCodeInternal, "internal error")
	}
	return fiber.Map{"moments": toMomentViews(list)}, nil
}

// commentListPending 待审核评论（6.8）：创建时间正序。
func (m *MCP) commentListPending(ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError) {
	list, err := m.comments.ListPending()
	if err != nil {
		return nil, mcpErr(fiber.StatusInternalServerError, mcpCodeInternal, "internal error")
	}
	return fiber.Map{"comments": toCommentViews(list)}, nil
}

// commentApprove 审核通过评论（6.9）：按宿主做失效声明。
func (m *MCP) commentApprove(ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError) {
	var in struct {
		ID string `json:"id"`
	}
	if merr := decodePayload(payload, &in); merr != nil {
		return nil, merr
	}
	id := mustID(in.ID)
	target, err := m.comments.Approve(id)
	if err != nil {
		return nil, classifyCommentError(err)
	}
	invalidateCommentHost(m.a, target)
	return fiber.Map{"id": strconv.FormatInt(id, 10), "status": comment.StatusApproved}, nil
}

// commentReject 驳回评论（6.10）：reason 必填 ≤200（对齐面板 rejectInput）。
// 注：与设计文档 §6.10 的偏差——Unit 3 实际签名是 Reject(id, reason)，故 payload 含 reason。
func (m *MCP) commentReject(ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError) {
	var in struct {
		ID     string `json:"id"`
		Reason string `json:"reason"`
	}
	if merr := decodePayload(payload, &in); merr != nil {
		return nil, merr
	}
	id := mustID(in.ID)
	target, err := m.comments.Reject(id, in.Reason)
	if err != nil {
		return nil, classifyCommentError(err)
	}
	// approved → rejected 是可见性变化，宿主页必须失效（统一走宿主分支）
	invalidateCommentHost(m.a, target)
	return fiber.Map{"id": strconv.FormatInt(id, 10), "status": comment.StatusRejected}, nil
}

// commentRecover 恢复被驳回评论（6.11）：Unit 3 语义为 rejected → approved（非 rejected 返回
// ErrInvalidState → validation）。
// 注：与设计文档 §6.11 的偏差——Unit 3 定稿直接回 approved（SetStatus(StatusApproved)），
// 非文档预期的 pending，成功 status 以实际代码为准。
func (m *MCP) commentRecover(ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError) {
	var in struct {
		ID string `json:"id"`
	}
	if merr := decodePayload(payload, &in); merr != nil {
		return nil, merr
	}
	id := mustID(in.ID)
	target, err := m.comments.Recover(id)
	if err != nil {
		return nil, classifyCommentError(err)
	}
	invalidateCommentHost(m.a, target)
	return fiber.Map{"id": strconv.FormatInt(id, 10), "status": comment.StatusApproved}, nil
}

// commentList 全站评论（6.12）：limit 缺省 100（对齐后台 ListAll(100)）。
func (m *MCP) commentList(ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError) {
	var in struct {
		Limit *int `json:"limit"`
	}
	if merr := decodePayload(payload, &in); merr != nil {
		return nil, merr
	}
	limit := 100
	if in.Limit != nil && *in.Limit > 0 {
		limit = *in.Limit
	}
	list, err := m.comments.ListAll(limit)
	if err != nil {
		return nil, mcpErr(fiber.StatusInternalServerError, mcpCodeInternal, "internal error")
	}
	return fiber.Map{"comments": toCommentViews(list)}, nil
}

// authorGet 读取作者个人页内容与资料（6.13）：content 缺省回退 + profile 含 email（作者本人可见）。
func (m *MCP) authorGet(ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError) {
	content, err := m.settings.Content()
	if err != nil {
		return nil, mcpErr(fiber.StatusInternalServerError, mcpCodeInternal, "internal error")
	}
	author, err := m.authorFn()
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, mcpErr(fiber.StatusNotFound, mcpCodeNotFound, "user not found")
		}
		return nil, mcpErr(fiber.StatusInternalServerError, mcpCodeInternal, "internal error")
	}
	return fiber.Map{
		"content": content,
		"profile": fiber.Map{
			"username":  author.Username,
			"role":      author.Role.String(),
			"bio":       author.Bio,
			"avatarUrl": author.AvatarURL,
			"email":     author.Email,
		},
	}, nil
}

// authorUpdatePayload 是 author.update 请求体：全字段可选（部分更新——只写传入的字段，
// 与面板"整包覆盖"不同；agent 只改某字段时不会误清空其余）。
type authorUpdatePayload struct {
	Paragraphs    *[]string                 `json:"paragraphs"`
	Skills        *[]settingsapp.Skill      `json:"skills"`
	Friends       *[]settingsapp.FriendLink `json:"friends"`
	Projects      *[]settingsapp.Project    `json:"projects"`
	Quotes        *[]settingsapp.Quote      `json:"quotes"`
	ShowcasePosts *[]int64                  `json:"showcasePosts"`
	Bio           *string                   `json:"bio"`
	AvatarURL     *string                   `json:"avatarUrl"`
	Username      *string                   `json:"username"`
}

// authorUpdate 部分更新作者页内容与资料（6.14）。
func (m *MCP) authorUpdate(ctx *fiber.Ctx, payload json.RawMessage) (any, *mcpError) {
	var in authorUpdatePayload
	if merr := decodePayload(payload, &in); merr != nil {
		return nil, merr
	}
	userID := mcpUserID(ctx)

	contentChanged := false
	set := func(fn func() error) *mcpError {
		if err := fn(); err != nil {
			return mcpErr(fiber.StatusInternalServerError, mcpCodeInternal, "internal error")
		}
		contentChanged = true
		return nil
	}
	if in.Paragraphs != nil {
		if merr := set(func() error { return m.settings.SetParagraphs(*in.Paragraphs) }); merr != nil {
			return nil, merr
		}
	}
	if in.Skills != nil {
		if merr := set(func() error { return m.settings.SetSkills(*in.Skills) }); merr != nil {
			return nil, merr
		}
	}
	if in.Friends != nil {
		if merr := set(func() error { return m.settings.SetFriends(*in.Friends) }); merr != nil {
			return nil, merr
		}
	}
	if in.Projects != nil {
		if merr := set(func() error { return m.settings.SetProjects(*in.Projects) }); merr != nil {
			return nil, merr
		}
	}
	if in.Quotes != nil {
		if merr := set(func() error { return m.settings.SetQuotes(*in.Quotes) }); merr != nil {
			return nil, merr
		}
	}
	if in.ShowcasePosts != nil {
		if merr := set(func() error { return m.settings.SetShowcasePosts(*in.ShowcasePosts) }); merr != nil {
			return nil, merr
		}
	}

	// 资料部分更新：只写传入字段，未传字段保持现值（UpdateProfile 是整包写入，需先取现值合并）
	profileChanged := false
	if in.Bio != nil || in.AvatarURL != nil {
		current, err := m.users.FindByID(userID)
		if err != nil {
			return nil, classifyUserError(err)
		}
		bio, avatar := current.Bio, current.AvatarURL
		if in.Bio != nil {
			bio = *in.Bio
		}
		if in.AvatarURL != nil {
			avatar = *in.AvatarURL
		}
		if err := m.users.UpdateProfile(userID, bio, avatar); err != nil {
			return nil, classifyUserError(err)
		}
		contentChanged = true
		profileChanged = true
	}

	// 改用户名：改名前先取旧用户名（旧路径失效需要，对齐 settings.go 用户名分支）
	usernameChanged, newUsername := false, ""
	if in.Username != nil {
		current, err := m.users.FindByID(userID)
		if err != nil {
			return nil, classifyUserError(err)
		}
		old := current.Username
		if err := m.users.UpdateUsername(userID, *in.Username); err != nil {
			return nil, classifyUserError(err)
		}
		usernameChanged, newUsername = true, *in.Username
		m.a.InvalidatePage("/")
		m.a.InvalidatePage("/author/" + old)
		m.a.InvalidatePage("/author/" + newUsername)
		// 动态页是 ISR 静态页，一并 DataChange 失效再生（对齐 settings.go 用户名分支）
		_ = m.a.DataChange("/moments")
	}

	if contentChanged {
		// content 与 profile 改动聚合：作者主页 + 首页（对齐 authorAdmin.go 失效组）
		m.a.InvalidatePage("/author/" + m.authorNameFn())
		m.a.InvalidatePage("/")
	}
	if profileChanged {
		// 资料（bio/头像）同展示于动态页 ISR 静态页，DataChange 失效再生（对齐 settings.go 资料分支）
		_ = m.a.DataChange("/moments")
	}

	if usernameChanged {
		return fiber.Map{"username": newUsername}, nil
	}
	return fiber.Map{"updated": true}, nil
}

// invalidateCommentHost 按评论宿主做失效声明（approve/reject/recover 共用）：
// 动态宿主 → /moments 失效；文章宿主 → declarePostsChanged。
func invalidateCommentHost(a *hybrid.App, target comment.Target) {
	if target.MomentID > 0 {
		_ = a.DataChange("/moments")
	} else {
		declarePostsChanged(a, target.PostID)
	}
}
