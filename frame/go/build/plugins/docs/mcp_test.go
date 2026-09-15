package docs

import (
	"encoding/json"
	"strings"
	"testing"

	"ven_hybird/build/plugin"
)

// newMCPTestService 起真用例服务（内存仓储），返回供 handler 直调。
func newMCPTestService() *Service { return newTestService() }

// call 以 JSON 字符串调 action，返回 (data 原始 JSON, actionErr)。
func call(svc *Service, action string, payload string) (json.RawMessage, *plugin.ActionError) {
	fn, ok := actionTable(svc)[action]
	if !ok {
		panic("unknown action in test: " + action)
	}
	data, aerr := fn(json.RawMessage(payload))
	if aerr != nil {
		return nil, aerr
	}
	raw, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return raw, nil
}

func actionTable(svc *Service) map[string]func(json.RawMessage) (any, *plugin.ActionError) {
	noop := func() {}
	return map[string]func(json.RawMessage) (any, *plugin.ActionError){
		"doc.create": wrap(mcpCreate, svc, noop),
		"doc.get":    wrapPlain(mcpGet, svc),
		"doc.list":   wrapPlain(mcpList, svc),
		"doc.tree":   wrapPlain(mcpTree, svc),
		"doc.update": wrap(mcpUpdate, svc, noop),
		"doc.delete": wrap(mcpDelete, svc, noop),
	}
}

func wrap(fn func(*Service, InvalidateFunc, json.RawMessage) (any, *plugin.ActionError), svc *Service, invalidate InvalidateFunc) func(json.RawMessage) (any, *plugin.ActionError) {
	return func(p json.RawMessage) (any, *plugin.ActionError) { return fn(svc, invalidate, p) }
}

func wrapPlain(fn func(*Service, json.RawMessage) (any, *plugin.ActionError), svc *Service) func(json.RawMessage) (any, *plugin.ActionError) {
	return func(p json.RawMessage) (any, *plugin.ActionError) { return fn(svc, p) }
}

func TestMCPDocCreate(t *testing.T) {
	svc := newMCPTestService()
	raw, aerr := call(svc, "doc.create", `{"title":"Attention","path":"ai/attention","content":"# A","tags":["ml"]}`)
	if aerr != nil {
		t.Fatalf("create: %+v", aerr)
	}
	var out struct {
		Doc docView `json:"doc"`
		ID  string  `json:"id"`
	}
	_ = json.Unmarshal(raw, &out)
	if out.ID == "" || out.Doc.ID == "" {
		t.Fatalf("ID 应字符串化：%s", raw)
	}
	if out.Doc.Path != "ai/attention" || len(out.Doc.Tags) != 1 {
		t.Fatalf("视图字段不对：%s", raw)
	}
	// 隐式补父后再取。
	raw, aerr = call(svc, "doc.get", `{"path":"ai"}`)
	if aerr != nil {
		t.Fatalf("get 隐式父: %+v", aerr)
	}
	if !containsJSON(raw, `"kind":"section"`) {
		t.Fatalf("隐式父应为 section：%s", raw)
	}
}

func TestMCPDocCreate_Validation(t *testing.T) {
	svc := newMCPTestService()
	cases := []struct {
		payload string
		code    string
	}{
		{`{"path":"x"}`, "validation"},                                  // 缺 title
		{`{"title":"T"}`, "validation"},                                 // 缺 path
		{`{"title":"T","path":"dup"}`, ""},                              // 首次成功
		{`{"title":"T2","path":"dup"}`, "validation"},                   // 重复路径
		{`{"title":"T","path":"deep/a/b/c/d/e/f/g/h/i"}`, "validation"}, // 超深
	}
	for i, tc := range cases {
		_, aerr := call(svc, "doc.create", tc.payload)
		if tc.code == "" {
			if aerr != nil {
				t.Fatalf("case %d 应成功: %+v", i, aerr)
			}
			continue
		}
		if aerr == nil || aerr.Code != tc.code {
			t.Fatalf("case %d: want %s got %+v", i, tc.code, aerr)
		}
	}
}

func TestMCPDocGet_NotFound(t *testing.T) {
	svc := newMCPTestService()
	_, aerr := call(svc, "doc.get", `{"path":"ghost"}`)
	if aerr == nil || aerr.Status != 404 || aerr.Code != "not_found" {
		t.Fatalf("应 404 not_found，得 %+v", aerr)
	}
}

func TestMCPDocGet_Children(t *testing.T) {
	svc := newMCPTestService()
	call(svc, "doc.create", `{"title":"N1","path":"s/n1"}`)
	call(svc, "doc.create", `{"title":"N2","path":"s/n2"}`)
	raw, aerr := call(svc, "doc.get", `{"path":"s"}`)
	if aerr != nil {
		t.Fatalf("get: %+v", aerr)
	}
	if !containsJSON(raw, `"children":[`) || !containsJSON(raw, "n1") || !containsJSON(raw, "n2") {
		t.Fatalf("应含子节点：%s", raw)
	}
}

func TestMCPDocList_Pagination(t *testing.T) {
	svc := newMCPTestService()
	for _, p := range []string{"a", "b", "c"} {
		call(svc, "doc.create", `{"title":"`+p+`","path":"`+p+`"}`)
	}
	raw, aerr := call(svc, "doc.list", `{"limit":2}`)
	if aerr != nil {
		t.Fatalf("list: %+v", aerr)
	}
	if !containsJSON(raw, `"total":3`) {
		t.Fatalf("total 应 3：%s", raw)
	}
	if !containsJSON(raw, `"docs":[`) || countOccurrences(raw, `"path"`) != 2 {
		t.Fatalf("limit 应生效：%s", raw)
	}
}

func TestMCPDocTree_IncludesDraft(t *testing.T) {
	svc := newMCPTestService()
	call(svc, "doc.create", `{"title":"D","path":"d","status":"draft"}`)
	raw, aerr := call(svc, "doc.tree", `{}`)
	if aerr != nil {
		t.Fatalf("tree: %+v", aerr)
	}
	if !containsJSON(raw, `"status":"draft"`) {
		t.Fatalf("MCP tree 应含 draft：%s", raw)
	}
}

func TestMCPDocUpdate_Partial(t *testing.T) {
	svc := newMCPTestService()
	call(svc, "doc.create", `{"title":"T","path":"p","content":"old","tags":["x"]}`)
	raw, aerr := call(svc, "doc.update", `{"path":"p","title":"T2"}`)
	if aerr != nil {
		t.Fatalf("update: %+v", aerr)
	}
	if !containsJSON(raw, `"title":"T2"`) || !containsJSON(raw, `"content":"old"`) {
		t.Fatalf("部分更新应保留 content：%s", raw)
	}
	// 状态切换 draft。
	raw, aerr = call(svc, "doc.update", `{"path":"p","status":"draft"}`)
	if aerr != nil || !containsJSON(raw, `"status":"draft"`) {
		t.Fatalf("状态更新: %+v %s", aerr, raw)
	}
}

func TestMCPDocDelete_RequiresRecursiveForChildren(t *testing.T) {
	svc := newMCPTestService()
	call(svc, "doc.create", `{"title":"N","path":"sec/n"}`)
	_, aerr := call(svc, "doc.delete", `{"path":"sec"}`)
	if aerr == nil || aerr.Code != "validation" {
		t.Fatalf("有子未 recursive 应 validation，得 %+v", aerr)
	}
	_, aerr = call(svc, "doc.delete", `{"path":"sec","recursive":true}`)
	if aerr != nil {
		t.Fatalf("级联删除: %+v", aerr)
	}
	_, aerr = call(svc, "doc.get", `{"path":"sec/n"}`)
	if aerr == nil || aerr.Status != 404 {
		t.Fatal("子节点应删除")
	}
}

func TestMCPDocCreate_InvalidPayload(t *testing.T) {
	svc := newMCPTestService()
	_, aerr := call(svc, "doc.create", `{not-json`)
	if aerr == nil || aerr.Code != "bad_request" {
		t.Fatalf("应 bad_request，得 %+v", aerr)
	}
}

func containsJSON(raw json.RawMessage, sub string) bool { return strings.Contains(string(raw), sub) }

func countOccurrences(raw json.RawMessage, sub string) int { return strings.Count(string(raw), sub) }
