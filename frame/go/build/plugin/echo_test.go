package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// echoPlugin 契约测试插件：验证插件经内核走完 Register（MCP action + Search provider）→
// Start → Stop 全流程（unit-6 §8 验收条款 2）。仅测试用，不进生产插件清单。
type echoPlugin struct {
	started bool
	stopped bool
}

func (e *echoPlugin) Meta() Meta {
	return Meta{
		Name:        "echo",
		Version:     "0.1.0",
		Description: "契约测试插件",
		PagePrefix:  []string{"/echo"},
		DefaultOn:   true,
	}
}

func (e *echoPlugin) Register(rt *Runtime) error {
	if rt.MCP == nil {
		return errors.New("echo: Runtime.MCP 未装配")
	}
	if rt.Search == nil {
		return errors.New("echo: Runtime.Search 未装配")
	}
	if err := rt.MCP.RegisterAction("echo.ping", func(payload json.RawMessage) (any, *ActionError) {
		var in struct {
			Msg string `json:"msg"`
		}
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, &ActionError{Status: http.StatusBadRequest, Code: "bad_request", Message: "invalid payload"}
		}
		return map[string]string{"pong": in.Msg}, nil
	}); err != nil {
		return err
	}
	return rt.Search.RegisterProvider(echoProvider{})
}

func (e *echoPlugin) Start() error { e.started = true; return nil }
func (e *echoPlugin) Stop() error  { e.stopped = true; return nil }

// echoProvider echo 插件的搜索贡献源。
type echoProvider struct{}

func (echoProvider) Name() string { return "echo" }
func (echoProvider) Search(ctx context.Context, q string, limit int) ([]SearchHit, error) {
	if strings.Contains(q, "hit") {
		return []SearchHit{{Title: "echo hit", URL: "/echo"}}, nil
	}
	return nil, nil
}

// fakeMCPRegistry / fakeSearchRegistry：内核单测用假注册面。
type fakeMCPRegistry struct {
	registered map[string]MCPActionFunc
	regErr     error
}

func (f *fakeMCPRegistry) RegisterAction(name string, fn MCPActionFunc) error {
	if f.regErr != nil {
		return f.regErr
	}
	if f.registered == nil {
		f.registered = make(map[string]MCPActionFunc)
	}
	if _, dup := f.registered[name]; dup {
		return errors.New("duplicate action")
	}
	f.registered[name] = fn
	return nil
}

type fakeSearchRegistry struct {
	providers map[string]SearchProvider
}

func (f *fakeSearchRegistry) RegisterProvider(p SearchProvider) error {
	if f.providers == nil {
		f.providers = make(map[string]SearchProvider)
	}
	if _, dup := f.providers[p.Name()]; dup {
		return errors.New("duplicate provider")
	}
	f.providers[p.Name()] = p
	return nil
}

func TestEchoPlugin_FullLifecycleViaBootstrap(t *testing.T) {
	echo := &echoPlugin{}
	mcp := &fakeMCPRegistry{}
	search := &fakeSearchRegistry{}
	settings := fakeSettings{}
	rt := &Runtime{
		MCP:      mcp,
		Search:   search,
		Settings: settings,
	}
	stopOrder, err := Bootstrap([]func() Plugin{func() Plugin { return echo }}, rt, nil)
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if !echo.started {
		t.Fatal("Startable 应已启动")
	}
	if len(mcp.registered) != 1 || mcp.registered["echo.ping"] == nil {
		t.Fatalf("echo.ping 应已注册，得 %v", mcp.registered)
	}
	if _, ok := search.providers["echo"]; !ok {
		t.Fatal("echo provider 应已注册")
	}
	// 贡献的 action 可调用。
	data, aerr := mcp.registered["echo.ping"](json.RawMessage(`{"msg":"hi"}`))
	if aerr != nil {
		t.Fatalf("echo.ping 调用失败: %+v", aerr)
	}
	if m, ok := data.(map[string]string); !ok || m["pong"] != "hi" {
		t.Fatalf("echo.ping 应回显，得 %v", data)
	}
	// 关停。
	Shutdown(stopOrder, nil)
	if !echo.stopped {
		t.Fatal("Stoppable 应已关停")
	}
}

func TestEchoPlugin_MissingCapabilityFailsFast(t *testing.T) {
	echo := &echoPlugin{}
	if _, err := Bootstrap([]func() Plugin{func() Plugin { return echo }}, &Runtime{Settings: fakeSettings{}}, nil); err == nil {
		t.Fatal("Runtime 能力面缺失应 fail-fast")
	}
}

func TestEchoPlugin_OffSwitchSkipsRegistration(t *testing.T) {
	echo := &echoPlugin{}
	settings := fakeSettings{"plugin.echo.enabled": "off"}
	rt := &Runtime{MCP: &fakeMCPRegistry{}, Search: &fakeSearchRegistry{}, Settings: settings}
	if _, err := Bootstrap([]func() Plugin{func() Plugin { return echo }}, rt, nil); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if echo.started || echo.stopped {
		t.Fatal("关闭的插件不应启动/关停")
	}
}
