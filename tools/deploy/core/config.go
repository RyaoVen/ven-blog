package core

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// 关键环境变量名。
const (
	EnvDSN         = "BLOG_MYSQL_DSN"
	EnvToken       = "VEN_INTERNAL_TOKEN"
	EnvNodePort    = "VEN_NODE_PORT"    // Node SSR worker 监听端口（默认 3000，Node 侧 config.ts 同读）
	EnvListenAddr  = "VEN_LISTEN_ADDR"  // Go 网关监听地址（形如 ":8080"，解析出端口，默认 8080）
	DefaultNodePort = 3000
	DefaultGoPort   = 8080
)

// Defaults 返回未配置时的默认值。
// 注意：VEN_INTERNAL_TOKEN 无默认值——网关侧（#166）已强制校验
// 空/development-token 拒绝启动，部署工具必须显式配置强令牌。
func Defaults() map[string]string {
	return map[string]string{}
}

// Config 是部署工具视角的配置视图（.env.local + 默认值）。
type Config struct {
	Values map[string]string // 仅 .env.local 中显式配置的键
}

// LoadConfig 读取仓库根 .env.local 并构造配置视图。
func LoadConfig(root string) (*Config, error) {
	f, err := LoadEnvFile(filepath.Join(root, ".env.local"))
	if err != nil {
		return nil, err
	}
	c := &Config{Values: map[string]string{}}
	for _, e := range f.Entries {
		if e.Kind == EntryKey {
			c.Values[e.Key] = e.Value
		}
	}
	return c, nil
}

// Get 取值：文件值优先，缺失时回退默认值。
func (c *Config) Get(key string) string {
	if v, ok := c.Values[key]; ok {
		return v
	}
	return Defaults()[key]
}

// Has 判断文件里是否显式配置了该键。
func (c *Config) Has(key string) bool {
	_, ok := c.Values[key]
	return ok
}

// Validate 校验必填项：BLOG_MYSQL_DSN 非空；VEN_INTERNAL_TOKEN 必填且拒绝默认值
// （网关侧 #166 已强制：空/development-token 拒绝启动，部署工具配置必须与之对齐）。
func (c *Config) Validate() error {
	if c.Get(EnvDSN) == "" {
		return fmt.Errorf("缺少必填配置 %s（运行 config 向导或 config --set %s=... 补上）", EnvDSN, EnvDSN)
	}
	if c.Get(EnvToken) == "" {
		return fmt.Errorf("缺少必填配置 %s（网关强制内部令牌，config --set %s=<强随机值> 补上）", EnvToken, EnvToken)
	}
	if c.Get(EnvToken) == "development-token" {
		return fmt.Errorf("%s 不能使用默认值 development-token（网关会拒绝启动），请配置强随机令牌", EnvToken)
	}
	return nil
}

// InternalToken 返回内部令牌（未配置时为空串——由 Validate 拦截，见下）。
func (c *Config) InternalToken() string { return c.Get(EnvToken) }

// NodePort 返回 Node worker 端口：VEN_NODE_PORT，空/非法值回退 3000（对齐 Node 侧 config.ts 语义）。
func (c *Config) NodePort() int {
	if raw := c.Get(EnvNodePort); raw != "" {
		if p, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && p > 0 && p < 65536 {
			return p
		}
	}
	return DefaultNodePort
}

// GoPort 返回 Go 网关端口：从 VEN_LISTEN_ADDR 解析（":8080"/"0.0.0.0:8080"/"127.0.0.1:8080" 均可），
// 空/无法解析时回退 8080。
func (c *Config) GoPort() int {
	if raw := c.Get(EnvListenAddr); raw != "" {
		if _, port, err := net.SplitHostPort(raw); err == nil {
			if p, perr := strconv.Atoi(port); perr == nil && p > 0 && p < 65536 {
				return p
			}
		}
	}
	return DefaultGoPort
}

// Env 返回注入子进程的环境：os.Environ() + .env.local（文件值覆盖系统环境）。
func (c *Config) Env() []string {
	env := dropEnvKeys(os.Environ(), c.Values)
	for k, v := range c.Values {
		env = append(env, k+"="+v)
	}
	return env
}

// dropEnvKeys 剔除与文件键同名的系统环境项，避免旧值残留。
func dropEnvKeys(env []string, keep map[string]string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		k := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			k = kv[:i]
		}
		if _, dup := keep[k]; !dup {
			out = append(out, kv)
		}
	}
	return out
}

// MaskedSummary 返回用于展示的配置摘要（敏感值打码，键按字母序）。
func (c *Config) MaskedSummary() []string {
	keys := make([]string, 0, len(c.Values))
	for k := range c.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%s=%s", k, MaskValue(k, c.Values[k])))
	}
	return out
}

// MaskValue 对敏感键打码：DSN 只保留账号与主机段（密码打码），其余敏感值全隐。
func MaskValue(key, value string) string {
	if value == "" {
		return "(未配置)"
	}
	upper := strings.ToUpper(key)
	if strings.Contains(upper, "DSN") {
		if at := strings.Index(value, "@"); at > 0 {
			if colon := strings.IndexByte(value, ':'); colon >= 0 && colon < at {
				return value[:colon+1] + "***" + value[at:]
			}
		}
		return "***"
	}
	if strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "TOKEN") ||
		strings.Contains(upper, "API_KEY") || strings.Contains(upper, "SECRET") {
		return "***"
	}
	return value
}

// ConfigSummary 输出当前配置摘要（敏感值打码）与 DSN 校验结果到 w。
func ConfigSummary(root string, w io.Writer) error {
	cfg, err := LoadConfig(root)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "配置来源: %s\n", filepath.Join(root, ".env.local"))
	for _, l := range cfg.MaskedSummary() {
		fmt.Fprintln(w, "  "+l)
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(w, "校验失败: "+err.Error())
		fmt.Fprintln(w, "提示: 用 config --set BLOG_MYSQL_DSN=... 补上，或删除 .env.local 后运行 config 向导")
		return nil
	}
	fmt.Fprintln(w, "校验通过: BLOG_MYSQL_DSN 已配置")
	return nil
}

// Root 返回仓库根：优先当前目录，否则向上查找（支持在 tools/deploy/ 下运行）。
// 找不到返回空串。
func Root() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if IsRoot(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// IsRoot 判断 dir 是否是仓库根（存在 frame/go 与 frame/node）。
func IsRoot(dir string) bool {
	_, errGo := os.Stat(filepath.Join(dir, "frame", "go"))
	_, errNode := os.Stat(filepath.Join(dir, "frame", "node"))
	return errGo == nil && errNode == nil
}
