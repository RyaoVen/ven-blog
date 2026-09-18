// Package docs 插件规范入口（unit-6 §5.3：每插件唯一规范文件 plugin.go）。
// New() 纯构造——全部 IO（开库/建表/注册）发生在 Register；启停时序由内核保证。
package docs

import (
	"database/sql"
	"log"

	"ven_hybird/build/interfaces"
	"ven_hybird/build/plugin"
)

// New 插件入口：组合根清单调用（plugins/docs.New）。
func New() plugin.Plugin { return &docsPlugin{} }

// docsPlugin 插件实现：数据层自包含（自建 DB 连接，Stop 关池——治理规则外部资源自治）。
type docsPlugin struct {
	db      *sql.DB
	svc     *Service
	webhook *WebhookDispatcher
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
	p.webhook = NewWebhookDispatcher(rt.Settings)
	// 页面注册先行（StaticPage 声明就绪后失效回调才可用——DataChange 对未声明模式会报错）。
	if err := registerPages(rt, p.svc); err != nil {
		_ = db.Close()
		return err
	}
	// MCP doc.* action：写操作成功后经 invalidate 失效 /docs 静态页并联动 SSE。
	invalidate := newInvalidate(rt)
	if rt.MCP != nil {
		if err := registerMCP(rt, p.svc, invalidate, &WriteHooks{Emit: p.webhook.Emit}); err != nil {
			_ = db.Close()
			return err
		}
	}
	// 搜索贡献源（unit-6 §5.4；scope=docs 单源/all 合并）。
	if rt.Search != nil {
		if err := rt.Search.RegisterProvider(NewDocSearchProvider(p.svc)); err != nil {
			_ = db.Close()
			return err
		}
	}
	// 首页/admin 仪表盘统计供数（unit-6 扩展点：插件供数、宿主消费，页面每次现取）。
	interfaces.RegisterDocsStats(func() *interfaces.DocsStats {
		books, err := p.svc.Bookshelf()
		if err != nil {
			return nil
		}
		published, err := p.svc.ListAllPublished()
		if err != nil {
			return nil
		}
		stats := &interfaces.DocsStats{Books: len(books)}
		for _, d := range published {
			stats.Docs++
			stats.TotalChars += len([]rune(d.Content))
			if d.UpdatedAt.After(stats.LastUpdated) {
				stats.LastUpdated = d.UpdatedAt
			}
		}
		return stats
	})
	// 后台管理页面与 API（#9）：页面注册先行（路由契约）。
	if err := registerAdminPages(rt, p.svc); err != nil {
		_ = db.Close()
		return err
	}
	if err := registerAdminAPI(rt, p.svc, invalidate, rt.Settings); err != nil {
		_ = db.Close()
		return err
	}
	return nil
}

// Start 实现 plugin.Startable：webhook 投递器 worker（未配置时空转）。
func (p *docsPlugin) Start() error { return p.webhook.Start() }

// Stop 优雅关停：webhook worker 退出 + 释放自建连接池（评审 P0 项）。
func (p *docsPlugin) Stop() error {
	if err := p.webhook.Stop(); err != nil {
		log.Printf("docs: webhook stop: %v", err)
	}
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}
