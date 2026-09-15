package interfaces

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"ven_hybird/build/plugin"
)

// extraEchoAction 测试用插件 action：回显 payload 里的 msg。
func extraEchoAction(payload json.RawMessage) (any, *plugin.ActionError) {
	var in struct {
		Msg string `json:"msg"`
	}
	if err := json.Unmarshal(payload, &in); err != nil {
		return nil, &plugin.ActionError{Status: http.StatusBadRequest, Code: mcpCodeBadRequest, Message: "invalid payload"}
	}
	if in.Msg == "" {
		return nil, &plugin.ActionError{Status: http.StatusNotFound, Code: mcpCodeNotFound, Message: "msg required"}
	}
	return map[string]string{"echo": in.Msg}, nil
}

func TestMCPRegisterAction_PluginActionDispatch(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	if err := env.mcp.RegisterAction("echo.hi", extraEchoAction); err != nil {
		t.Fatalf("RegisterAction: %v", err)
	}
	resp, body := env.call(t, "ven_valid", `{"action":"echo.hi","payload":{"msg":"hello"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	if !containsStr(body, `"echo":"hello"`) {
		t.Fatalf("应回显 msg，得 %s", body)
	}
}

func TestMCPRegisterAction_PluginActionErrorMapping(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	if err := env.mcp.RegisterAction("echo.hi", extraEchoAction); err != nil {
		t.Fatalf("RegisterAction: %v", err)
	}
	// validation → 400；not_found 分支同验证错误协议转换。
	resp, body := env.call(t, "ven_valid", `{"action":"echo.hi","payload":{}}`)
	if resp.StatusCode != http.StatusNotFound || !containsStr(body, "msg required") {
		t.Fatalf("ActionError 应原样映射，得 %d %s", resp.StatusCode, body)
	}
}

func TestMCPRegisterAction_DuplicateRejected(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	if err := env.mcp.RegisterAction("echo.hi", extraEchoAction); err != nil {
		t.Fatalf("第一次注册应成功: %v", err)
	}
	if err := env.mcp.RegisterAction("echo.hi", extraEchoAction); err == nil {
		t.Fatal("重复注册应拒绝")
	}
}

func TestMCPRegisterAction_BuiltinNameRejected(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	if err := env.mcp.RegisterAction("post.create", extraEchoAction); err == nil {
		t.Fatal("与内置 action 重名应拒绝")
	}
}

func TestMCPRegisterAction_BadNames(t *testing.T) {
	for _, name := range []string{"", "noprefix", ".create", "echo.", "Echo.x", "echo-.x", "-e.x", "e--x.y", "e x.y"} {
		env := newMCPTestEnv(t, nil)
		if err := env.mcp.RegisterAction(name, extraEchoAction); err == nil {
			t.Fatalf("非法 action 名 %q 应拒绝", name)
		}
	}
}

func TestMCPRegisterAction_NilFnRejected(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	if err := env.mcp.RegisterAction("echo.hi", nil); err == nil {
		t.Fatal("nil 处理函数应拒绝")
	}
}

// containsStr 子串断言辅助。
func containsStr(s, sub string) bool { return strings.Contains(s, sub) }
