package docs

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ven_hybird/build/plugin"
)

// registerMCP 注册 doc.* action（unit-7 §4；unit-6 §5.2 经 Runtime.MCP）。
// 全部 action 为 author 语义（/api/mcp 网关 key 鉴权已保证）；
// ID 一律字符串化；写操作非幂等（失败先 doc.list 查证再重试，契约与 post 一致）。
func registerMCP(rt *plugin.Runtime, svc *Service, invalidate InvalidateFunc, hooks *WriteHooks) error {
	actions := map[string]plugin.MCPActionFunc{
		"doc.create": func(payload json.RawMessage) (any, *plugin.ActionError) {
			return mcpCreate(svc, invalidate, hooks, payload)
		},
		"doc.get":  func(payload json.RawMessage) (any, *plugin.ActionError) { return mcpGet(svc, payload) },
		"doc.list": func(payload json.RawMessage) (any, *plugin.ActionError) { return mcpList(svc, payload) },
		"doc.tree": func(payload json.RawMessage) (any, *plugin.ActionError) { return mcpTree(svc, payload) },
		"doc.move": func(payload json.RawMessage) (any, *plugin.ActionError) {
			return mcpMove(svc, invalidate, hooks, payload)
		},
		"doc.update": func(payload json.RawMessage) (any, *plugin.ActionError) {
			return mcpUpdate(svc, invalidate, hooks, payload)
		},
		"doc.delete": func(payload json.RawMessage) (any, *plugin.ActionError) {
			return mcpDelete(svc, invalidate, hooks, payload)
		},
	}
	for name, fn := range actions {
		if err := rt.MCP.RegisterAction(name, fn); err != nil {
			return err
		}
	}
	return nil
}

// docView 对外文档视图：ID 字符串化（unit-2 契约）。
type docView struct {
	ID        string   `json:"id"`
	ParentID  string   `json:"parentId"`
	Slug      string   `json:"slug"`
	Path      string   `json:"path"`
	Kind      Kind     `json:"kind"`
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	Content   string   `json:"content,omitempty"`
	Tags      []string `json:"tags"`
	SortOrder int      `json:"sortOrder"`
	Status    Status   `json:"status"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

// toView 实体转视图。
func toView(d *Doc, withContent bool) docView {
	v := docView{
		ID:        fmt.Sprintf("%d", d.ID),
		ParentID:  fmt.Sprintf("%d", d.ParentID),
		Slug:      d.Slug,
		Path:      d.Path,
		Kind:      d.Kind,
		Title:     d.Title,
		Summary:   d.Summary,
		Tags:      d.Tags,
		SortOrder: d.SortOrder,
		Status:    d.Status,
		CreatedAt: d.CreatedAt.UTC().Format(timeRFC3339),
		UpdatedAt: d.UpdatedAt.UTC().Format(timeRFC3339),
	}
	if withContent {
		v.Content = d.Content
	}
	return v
}

// timeRFC3339 时间格式（agent 解析友好）。
const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

// decode 解 payload 到结构体。
func decode(payload json.RawMessage, v any) *plugin.ActionError {
	if err := json.Unmarshal(payload, v); err != nil {
		return badRequest("invalid payload")
	}
	return nil
}

func badRequest(msg string) *plugin.ActionError {
	return &plugin.ActionError{Status: http.StatusBadRequest, Code: "bad_request", Message: msg}
}

func validation(err error) *plugin.ActionError {
	return &plugin.ActionError{Status: http.StatusBadRequest, Code: "validation", Message: err.Error()}
}

// mapErr 领域错误 → 协议错误（unit-7 §4：复用 mcpError 语义）。
func mapErr(err error) *plugin.ActionError {
	switch {
	case errors.Is(err, ErrNotFound):
		return &plugin.ActionError{Status: http.StatusNotFound, Code: "not_found", Message: "doc not found"}
	case errors.Is(err, ErrDuplicatePath), errors.Is(err, ErrHasChildren),
		errors.Is(err, ErrInvalidSlug), errors.Is(err, ErrTooDeep),
		errors.Is(err, ErrInvalidKind), errors.Is(err, ErrInvalidStatus),
		errors.Is(err, ErrTooManyTags), errors.Is(err, ErrTagTooLong):
		return validation(err)
	case err == nil:
		return nil
	default:
		// 标题长度等用例层错误也归 validation（消息可读）。
		msg := err.Error()
		if strings.Contains(msg, "required") || strings.Contains(msg, "too long") ||
			strings.Contains(msg, "不能作为父级") {
			return validation(err)
		}
		return &plugin.ActionError{Status: http.StatusInternalServerError, Code: "internal", Message: "internal error"}
	}
}

// WriteHooks 写操作旁路钩子：失效静态页 + 出站 webhook（接口层职责集中）。
type WriteHooks struct {
	Invalidate InvalidateFunc
	Emit       func(event, path string)
}

// emit 安全触发 webhook（nil 安全）。
func (h *WriteHooks) emit(event, path string) {
	if h != nil && h.Emit != nil {
		h.Emit(event, path)
	}
}

// mcpCreate doc.create。
func mcpCreate(svc *Service, invalidate InvalidateFunc, hooks *WriteHooks, payload json.RawMessage) (any, *plugin.ActionError) {
	var in struct {
		Title   string   `json:"title"`
		Path    string   `json:"path"`
		Kind    Kind     `json:"kind"`
		Content *string  `json:"content"`
		Summary *string  `json:"summary"`
		Tags    []string `json:"tags"`
		Order   *int     `json:"order"`
		Status  Status   `json:"status"`
	}
	if err := decode(payload, &in); err != nil {
		return nil, err
	}
	if in.Title == "" {
		return nil, validation(errors.New("title is required"))
	}
	if in.Path == "" {
		return nil, validation(errors.New("path is required"))
	}
	doc, err := svc.Create(CreateInput{
		Title: in.Title, Path: in.Path, Kind: in.Kind, Content: in.Content,
		Summary: in.Summary, Tags: in.Tags, Order: in.Order, Status: in.Status,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	invalidate()
	hooks.emit("doc.created", doc.Path)
	return map[string]any{"doc": toView(doc, true), "id": docViewID(doc)}, nil
}

// docViewID ID 字符串化。
func docViewID(d *Doc) string { return fmt.Sprintf("%d", d.ID) }

// mcpGet doc.get：节点 + 直接子节点。
func mcpGet(svc *Service, payload json.RawMessage) (any, *plugin.ActionError) {
	var in struct {
		Path string `json:"path"`
	}
	if err := decode(payload, &in); err != nil {
		return nil, err
	}
	if in.Path == "" {
		return nil, validation(errors.New("path is required"))
	}
	res, err := svc.Get(in.Path)
	if err != nil {
		return nil, mapErr(err)
	}
	children := make([]docView, 0, len(res.Children))
	for _, c := range res.Children {
		children = append(children, toView(c, false))
	}
	return map[string]any{"doc": toView(res.Doc, true), "children": children}, nil
}

// mcpList doc.list。
func mcpList(svc *Service, payload json.RawMessage) (any, *plugin.ActionError) {
	var in struct {
		Path      string `json:"path"`
		Recursive bool   `json:"recursive"`
		Limit     int    `json:"limit"`
		Offset    int    `json:"offset"`
		Since     string `json:"since"` // ISO8601（增量拉取，M5）
	}
	if err := decode(payload, &in); err != nil {
		return nil, err
	}
	var since *time.Time
	if in.Since != "" {
		t, err := time.Parse(time.RFC3339, in.Since)
		if err != nil {
			return nil, validation(errors.New("since must be RFC3339/ISO8601"))
		}
		since = &t
	}
	docs, total, err := svc.List(ListInput{Path: in.Path, Recursive: in.Recursive, Limit: in.Limit, Offset: in.Offset, Since: since})
	if err != nil {
		return nil, mapErr(err)
	}
	views := make([]docView, 0, len(docs))
	for _, d := range docs {
		views = append(views, toView(d, false))
	}
	return map[string]any{"docs": views, "total": total}, nil
}

// mcpTree doc.tree：全量导航树（MCP 侧含 draft）。
func mcpTree(svc *Service, payload json.RawMessage) (any, *plugin.ActionError) {
	tree, err := svc.Tree(false)
	if err != nil {
		return nil, mapErr(err)
	}
	return map[string]any{"tree": tree}, nil
}

// mcpMove doc.move：移动/改名/重排并级联子树 path（M5）。
func mcpMove(svc *Service, invalidate InvalidateFunc, hooks *WriteHooks, payload json.RawMessage) (any, *plugin.ActionError) {
	var in struct {
		Path      string `json:"path"`
		NewParent string `json:"newParent"`
		NewSlug   string `json:"newSlug"`
		Order     *int   `json:"order"`
	}
	if err := decode(payload, &in); err != nil {
		return nil, err
	}
	if in.Path == "" {
		return nil, validation(errors.New("path is required"))
	}
	doc, err := svc.Move(MoveInput{Path: in.Path, NewParent: in.NewParent, NewSlug: in.NewSlug, Order: in.Order})
	if err != nil {
		return nil, mapErr(err)
	}
	invalidate()
	hooks.emit("doc.moved", doc.Path)
	return map[string]any{"doc": toView(doc, false), "moved": true}, nil
}

// mcpUpdate doc.update（部分更新：指针字段判存在）。
func mcpUpdate(svc *Service, invalidate InvalidateFunc, hooks *WriteHooks, payload json.RawMessage) (any, *plugin.ActionError) {
	var in struct {
		Path    string   `json:"path"`
		Title   *string  `json:"title"`
		Content *string  `json:"content"`
		Summary *string  `json:"summary"`
		Tags    []string `json:"tags"`
		Order   *int     `json:"order"`
		Status  *Status  `json:"status"`
	}
	if err := decode(payload, &in); err != nil {
		return nil, err
	}
	if in.Path == "" {
		return nil, validation(errors.New("path is required"))
	}
	doc, err := svc.Update(UpdateInput{
		Path: in.Path, Title: in.Title, Content: in.Content, Summary: in.Summary,
		Tags: in.Tags, Order: in.Order, Status: in.Status,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	invalidate()
	hooks.emit("doc.updated", doc.Path)
	return map[string]any{"doc": toView(doc, true), "updated": true}, nil
}

// mcpDelete doc.delete。
func mcpDelete(svc *Service, invalidate InvalidateFunc, hooks *WriteHooks, payload json.RawMessage) (any, *plugin.ActionError) {
	var in struct {
		Path      string `json:"path"`
		Recursive bool   `json:"recursive"`
	}
	if err := decode(payload, &in); err != nil {
		return nil, err
	}
	if in.Path == "" {
		return nil, validation(errors.New("path is required"))
	}
	if err := svc.Delete(DeleteInput{Path: in.Path, Recursive: in.Recursive}); err != nil {
		return nil, mapErr(err)
	}
	invalidate()
	hooks.emit("doc.deleted", in.Path)
	return map[string]any{"deleted": true}, nil
}
