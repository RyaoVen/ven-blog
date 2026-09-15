// Package docs 插件规范入口（unit-6 §5.3：每插件唯一规范文件 plugin.go）。
// New() 纯构造——全部 IO（开库/建表/注册）发生在 Register；启停时序由内核保证。
package docs

import (
	"database/sql"

	"ven_hybird/build/plugin"
)

// New 插件入口：组合根清单调用（plugins/docs.New）。
func New() plugin.Plugin { return &docsPlugin{} }

// docsPlugin 插件实现：数据层自包含（自建 DB 连接，Stop 关池——治理规则外部资源自治）。
type docsPlugin struct {
	db  *sql.DB
	svc *Service
}

// Meta 元数据：路由前缀所有权 /docs；settings 键 plugin.docs.enabled 控制启停。
func (p *docsPlugin) Meta() plugin.Meta {
	return plugin.Meta{
		Name:        "docs",
		Version:     "0.1.0",
		Description: "树形文档模块（笔记、项目文档）",
		PagePrefix:  []string{"/docs"},
		DefaultOn:   true,
	}
}

// Register 接线：自建 DB 连接 + 幂等建表 + 用例服务（页面/API/MCP 注册随里程碑追加）。
func (p *docsPlugin) Register(rt *plugin.Runtime) error {
	db, err := OpenDB()
	if err != nil {
		return err
	}
	if err := ensureTable(db); err != nil {
		_ = db.Close()
		return err
	}
	p.db = db
	p.svc = NewService(NewDocRepository(db))
	// MCP doc.* action（#7）；失效声明 DataChange("/docs/*") 随 #8 页面注册一并启用
	// （StaticPage 未声明时 DataChange 会报错，页面先行是框架约束）。
	if rt.MCP != nil {
		if err := registerMCP(rt, p.svc); err != nil {
			_ = db.Close()
			return err
		}
	}
	return nil
}

// Stop 优雅关停：释放自建连接池（评审 P0 项）。
func (p *docsPlugin) Stop() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}
