// Package plugin 提供业务插件的契约与注册内核（docs/agent-design/unit-6-plugin-system.md）。
//
// 插件 = 自包含业务模块：编译期进二进制、启动期经 Bootstrap 校验/启停/装配。
// 治理规则：路由前缀所有权（PagePrefix）、MCP action 命名前缀（<plugin>.）、
// 窄接口 Runtime（禁止裸传 *sql.DB 等宽依赖）、显式清单注册（禁 init 自注册/反射扫描）。
//
// 入口约定：每插件唯一规范文件 plugin.go，入口 New() 必须纯构造（无 IO/无全局副作用），
// 全部接线发生在 Register——内核时序是「构造 → 读 Meta → 查 settings 启停 → 才调 Register」，
// New 自带副作用会使启停治理失效。
package plugin

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"ven_hybird/hybrid"
)

// Meta 插件元数据，Register 前由内核校验。
type Meta struct {
	Name        string // 唯一标识，kebab-case
	Version     string // semver
	Description string
	PagePrefix  []string // 页面路由前缀所有权（如 "/docs"），内核做冲突校验
	Depends     []string // 依赖的插件名（仅拓扑排序，不做版本区间）
	DefaultOn   bool     // settings 未配置启停键时的默认值
}

// Runtime 受控能力面：内核装配，窄接口注入（对齐 unit-2 KeyAuthenticator 先例）。
// 插件不得越过 Runtime 直取宿主内部；字段可为 nil（宿主未装配该能力时）。
type Runtime struct {
	App        *hybrid.App    // 框架注册面（Page/StaticPage/API/Role）
	MCP        MCPRegistry    // mcp action 注册面
	Search     SearchRegistry // 搜索 provider 注册面
	Settings   SettingsStore  // 插件配置命名空间（plugin.<name>.* 自管）
	DataChange DataChangeFunc // 页面失效辅助（DataChange 同形）
	Logger     *log.Logger    // nil 时由使用方兜底
}

// MCPRegistry 插件贡献 mcp action 的注册面（由 interfaces 包的 MCP 实现）。
// name 必须 "<plugin>." 前缀（如 "doc.create"），重名注册期拒绝。
type MCPRegistry interface {
	RegisterAction(name string, fn MCPActionFunc) error
}

// MCPActionFunc 插件 action 处理签名：payload 已归一为非空对象 JSON；
// 返回 data（成功，ID 一律字符串化）或 *ActionError。
type MCPActionFunc func(payload json.RawMessage) (any, *ActionError)

// ActionError action 处理错误，与 /api/mcp 网关错误契约同形。
type ActionError struct {
	Status  int    // HTTP 状态码（400/404/500…）
	Code    string // 业务码（validation/not_found/bad_request/internal…）
	Message string
}

// SearchRegistry 搜索结果贡献注册面（由 interfaces 包的聚合器实现）。
type SearchRegistry interface {
	RegisterProvider(p SearchProvider) error
}

// SearchProvider 搜索结果贡献者；Name 即对外 scope 取值（= 插件名，如 "docs"）。
// ctx 为请求作用域，实现必须尊重其超时/取消。
type SearchProvider interface {
	Name() string
	Search(ctx context.Context, q string, limit int) ([]SearchHit, error)
}

// SearchHit 单条搜索结果。
type SearchHit struct {
	Title     string
	Summary   string
	URL       string
	UpdatedAt time.Time
}

// SettingsStore 插件配置键值读写。不存在返回空串与 nil 错误（与 setting.Repository 同形）。
// 插件自管 plugin.<name>.* 命名空间；越界访问属评审问题，内核不做路径强制。
type SettingsStore interface {
	Get(key string) (string, error)
	Set(key, value string) error
}

// DataChangeFunc 页面失效辅助，与 hybrid.App.DataChange 同形。
type DataChangeFunc func(pattern string, params ...string) error

// Plugin 插件契约。实现方在其规范文件 plugin.go 提供 New() 入口。
type Plugin interface {
	Meta() Meta
	Register(rt *Runtime) error
}

// Startable worker 类插件可选实现（goroutine + ticker，参照 moderator 先例；不阻塞注册）。
type Startable interface {
	Start() error
}

// Stoppable 优雅关停；宿主关停时逆序调用。必须释放自建资源（如 DB 连接池）——评审 P0。
type Stoppable interface {
	Stop() error
}
