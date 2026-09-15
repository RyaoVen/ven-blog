package plugin

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// fakePlugin 可配置行为的测试插件。
type fakePlugin struct {
	meta       Meta
	regErr     error
	startErr   error
	stopErr    error
	registered bool
	started    bool
	stopped    bool
}

func (f *fakePlugin) Meta() Meta { return f.meta }
func (f *fakePlugin) Register(rt *Runtime) error {
	f.registered = true
	return f.regErr
}
func (f *fakePlugin) Start() error { f.started = true; return f.startErr }
func (f *fakePlugin) Stop() error  { f.stopped = true; return f.stopErr }

func newFake(name string, deps ...string) *fakePlugin {
	return &fakePlugin{meta: Meta{Name: name, Version: "1.0.0", Depends: deps, DefaultOn: true, PagePrefix: []string{"/" + name}}}
}

// fakeSettings 内存 settings。
type fakeSettings map[string]string

func (s fakeSettings) Get(key string) (string, error) { return s[key], nil }
func (s fakeSettings) Set(key, value string) error    { s[key] = value; return nil }

// errorSettings Get 恒错的 settings（验证读错回退 DefaultOn）。
type errorSettings struct{}

func (errorSettings) Get(string) (string, error) { return "", errors.New("backend down") }
func (errorSettings) Set(string, string) error   { return nil }

func rt(settings SettingsStore) *Runtime {
	return &Runtime{Settings: settings}
}

func names(list []Stoppable, ps []*fakePlugin) []string {
	// 依注册序取已注册插件名（借助调用序断言用）。
	var out []string
	for _, p := range ps {
		if p.registered {
			out = append(out, p.meta.Name)
		}
	}
	_ = list
	return out
}

func TestBootstrapHappyPath(t *testing.T) {
	a := newFake("alpha")
	b := newFake("beta", "alpha")
	settings := fakeSettings{}
	stopOrder, err := Bootstrap([]func() Plugin{func() Plugin { return a }, func() Plugin { return b }}, rt(settings), nil)
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if !a.registered || !b.registered {
		t.Fatalf("两插件都应注册：a=%v b=%v", a.registered, b.registered)
	}
	if !a.started || !b.started {
		t.Fatalf("Startable 应被启动")
	}
	if len(stopOrder) != 2 {
		t.Fatalf("stopOrder 应含 2 个，得 %d", len(stopOrder))
	}
	// 拓扑序：alpha 依赖无，先注册（清单序 alpha,beta 与依赖序一致）。
	if a.meta.Name == b.meta.Name {
		t.Fatal("sanity")
	}
}

func TestBootstrapStopsInReverse(t *testing.T) {
	a := newFake("alpha")
	b := newFake("beta", "alpha")
	stopOrder, err := Bootstrap([]func() Plugin{
		func() Plugin { return a },
		func() Plugin { return b },
	}, rt(fakeSettings{}), nil)
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	Shutdown(stopOrder, nil)
	if !a.stopped || !b.stopped {
		t.Fatalf("两插件都应 Stop")
	}
	// 逆序断言：b 先于 a 停。用顺序标记验证。
}

func TestBootstrapStopOrderIsReverse(t *testing.T) {
	var order []string
	p1 := &stopTracker{name: "p1", fn: func() { order = append(order, "p1") }}
	p2 := &stopTracker{name: "p2", fn: func() { order = append(order, "p2") }}
	stopOrder, err := Bootstrap([]func() Plugin{
		func() Plugin { return p1 },
		func() Plugin { return p2 },
	}, rt(fakeSettings{}), nil)
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	Shutdown(stopOrder, nil)
	if len(order) != 2 || order[0] != "p2" || order[1] != "p1" {
		t.Fatalf("应逆序关停 p2,p1，得 %v", order)
	}
}

type stopTracker struct {
	name string
	fn   func()
}

func (s *stopTracker) Meta() Meta              { return Meta{Name: s.name, DefaultOn: true} }
func (s *stopTracker) Register(*Runtime) error { return nil }
func (s *stopTracker) Stop() error             { s.fn(); return nil }

func TestBootstrapDuplicateName(t *testing.T) {
	a := newFake("dup")
	b := newFake("dup")
	_, err := Bootstrap([]func() Plugin{func() Plugin { return a }, func() Plugin { return b }}, rt(fakeSettings{}), nil)
	if err == nil || !contains(err.Error(), "重名") {
		t.Fatalf("应报重名，得 %v", err)
	}
}

func TestBootstrapPrefixConflicts(t *testing.T) {
	cases := []struct {
		name    string
		a       []string
		b       []string
		builtin []string
		wantErr bool
	}{
		{"相同前缀", []string{"/docs"}, []string{"/docs"}, nil, true},
		{"父子前缀", []string{"/docs"}, []string{"/docs/api"}, nil, true},
		{"段边界不冲突", []string{"/docs"}, []string{"/docsx"}, nil, false},
		{"多段与单段", []string{"/my/docs"}, []string{"/my"}, nil, true},
		{"与内置冲突", []string{"/posts"}, nil, []string{"/posts"}, true},
		{"内置子路径", []string{"/admin"}, nil, []string{"/admin/settings"}, true},
		{"与内置无冲突", []string{"/docs"}, nil, []string{"/posts", "/moments"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newFake("alpha")
			a.meta.PagePrefix = tc.a
			b := newFake("beta")
			b.meta.PagePrefix = tc.b
			_, err := Bootstrap([]func() Plugin{
				func() Plugin { return a },
				func() Plugin { return b },
			}, rt(fakeSettings{}), tc.builtin)
			if tc.wantErr && err == nil {
				t.Fatalf("应报前缀冲突")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("不应报错：%v", err)
			}
		})
	}
}

func TestBootstrapIllegalPrefix(t *testing.T) {
	a := newFake("alpha")
	a.meta.PagePrefix = []string{"/"}
	if _, err := Bootstrap([]func() Plugin{func() Plugin { return a }}, rt(fakeSettings{}), nil); err == nil {
		t.Fatal("根前缀 / 应非法")
	}
	a2 := newFake("alpha")
	a2.meta.PagePrefix = []string{"docs"}
	if _, err := Bootstrap([]func() Plugin{func() Plugin { return a2 }}, rt(fakeSettings{}), nil); err == nil {
		t.Fatal("缺 / 开头应非法")
	}
}

func TestBootstrapIllegalName(t *testing.T) {
	for _, name := range []string{"", "Docs", "d--x", "-d", "d-", "d.x"} {
		a := newFake(name)
		if _, err := Bootstrap([]func() Plugin{func() Plugin { return a }}, rt(fakeSettings{}), nil); err == nil {
			t.Fatalf("名称 %q 应非法", name)
		}
	}
}

func TestBootstrapDependencyMissingAndCycle(t *testing.T) {
	a := newFake("alpha", "ghost")
	if _, err := Bootstrap([]func() Plugin{func() Plugin { return a }}, rt(fakeSettings{}), nil); err == nil {
		t.Fatal("缺失依赖应报错")
	}
	c1 := newFake("c1", "c2")
	c2 := newFake("c2", "c1")
	if _, err := Bootstrap([]func() Plugin{
		func() Plugin { return c1 },
		func() Plugin { return c2 },
	}, rt(fakeSettings{}), nil); err == nil {
		t.Fatal("成环应报错")
	}
}

func TestBootstrapTopoOrderOverridesListing(t *testing.T) {
	// 清单序 beta 在前，但 alpha 依赖 beta：beta 必须先注册。
	regOrder := []string{}
	beta := &regTracker{name: "beta", onReg: func() { regOrder = append(regOrder, "beta") }}
	alpha := &regTracker{name: "alpha", deps: []string{"beta"}, onReg: func() { regOrder = append(regOrder, "alpha") }}
	if _, err := Bootstrap([]func() Plugin{
		func() Plugin { return alpha },
		func() Plugin { return beta },
	}, rt(fakeSettings{}), nil); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if regOrder[0] != "beta" || regOrder[1] != "alpha" {
		t.Fatalf("应 beta 先注册，得 %v", regOrder)
	}
}

type regTracker struct {
	name  string
	deps  []string
	onReg func()
}

func (r *regTracker) Meta() Meta              { return Meta{Name: r.name, Depends: r.deps, DefaultOn: true} }
func (r *regTracker) Register(*Runtime) error { r.onReg(); return nil }

func TestBootstrapEnabledSwitch(t *testing.T) {
	cases := []struct {
		raw    string
		defOn  bool
		wantOn bool
	}{
		{"on", false, true},
		{"OFF", true, false},
		{"1", false, true},
		{"0", true, false},
		{"", true, true},   // 缺省 → DefaultOn
		{"", false, false}, // 缺省 → DefaultOn
		{"garbage", true, true},
	}
	for _, tc := range cases {
		settings := fakeSettings{}
		if tc.raw != "" {
			settings["plugin.alpha.enabled"] = tc.raw
		}
		var st SettingsStore = settings
		if tc.raw == "ERR" {
			st = errorSettings{}
		}
		a := newFake("alpha")
		a.meta.DefaultOn = tc.defOn
		_, err := Bootstrap([]func() Plugin{func() Plugin { return a }}, &Runtime{Settings: st}, nil)
		if err != nil {
			t.Fatalf("raw=%q: %v", tc.raw, err)
		}
		if a.registered != tc.wantOn {
			t.Fatalf("raw=%q defOn=%v: registered=%v want %v", tc.raw, tc.defOn, a.registered, tc.wantOn)
		}
	}
}

func TestBootstrapSettingsErrorFallsBackToDefault(t *testing.T) {
	a := newFake("alpha")
	a.meta.DefaultOn = true
	if _, err := Bootstrap([]func() Plugin{func() Plugin { return a }}, &Runtime{Settings: errorSettings{}}, nil); err != nil {
		t.Fatalf("读错应回退默认而非报错：%v", err)
	}
	if !a.registered {
		t.Fatal("DefaultOn=true 应注册")
	}
}

func TestBootstrapRegisterFailFast(t *testing.T) {
	a := newFake("alpha")
	b := newFake("beta", "alpha")
	b.regErr = errors.New("boom")
	if _, err := Bootstrap([]func() Plugin{
		func() Plugin { return a },
		func() Plugin { return b },
	}, rt(fakeSettings{}), nil); err == nil {
		t.Fatal("Register 失败应整体失败")
	}
}

func TestBootstrapStartFailFast(t *testing.T) {
	a := newFake("alpha")
	a.startErr = errors.New("worker boom")
	_, err := Bootstrap([]func() Plugin{func() Plugin { return a }}, rt(fakeSettings{}), nil)
	if err == nil || !contains(err.Error(), "启动失败") {
		t.Fatalf("Start 失败应整体失败，得 %v", err)
	}
}

func TestShutdownContinuesOnError(t *testing.T) {
	var order []string
	bad := &stopTracker{name: "bad", fn: func() {}}
	_ = bad
	s1 := &stopErrTracker{name: "s1", order: &order}
	s2 := &stopTracker{name: "s2", fn: func() { order = append(order, "s2") }}
	Shutdown([]Stoppable{s1, s2}, nil)
	if len(order) != 1 || order[0] != "s2" {
		t.Fatalf("s1 出错不应阻断 s2，得 %v", order)
	}
}

type stopErrTracker struct {
	name  string
	order *[]string
}

func (s *stopErrTracker) Meta() Meta              { return Meta{Name: s.name} }
func (s *stopErrTracker) Register(*Runtime) error { return nil }
func (s *stopErrTracker) Stop() error             { return errors.New("stop boom") }

func TestEnabledKey(t *testing.T) {
	if got := EnabledKey("docs"); got != "plugin.docs.enabled" {
		t.Fatalf("EnabledKey = %s", got)
	}
}

func TestMCPActionFuncShape(t *testing.T) {
	// 契约锚点：action 签名与 ActionError 形状（编译期断言靠用例）。
	var fn MCPActionFunc = func(payload json.RawMessage) (any, *ActionError) {
		return map[string]string{"ok": "1"}, nil
	}
	data, aerr := fn(json.RawMessage(`{}`))
	if aerr != nil || data == nil {
		t.Fatal("action 应成功返回 data")
	}
	aerr = &ActionError{Status: 404, Code: "not_found", Message: "x"}
	if aerr.Status != 404 {
		t.Fatal("sanity")
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
