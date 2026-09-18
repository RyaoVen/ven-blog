// docsstats —— docs 模块统计注入点（unit-6 扩展点模式：插件供数、宿主页消费）。
// 首页 stats 面板与 /admin 数据面板经此获得 docs 维度计数；docs 插件未启用时为零值安全。
package interfaces

import (
	"sync"
	"time"
)

// DocsStats docs 模块统计快照。
type DocsStats struct {
	Books       int       // 顶层册数（书架上的书）
	Docs        int       // 文档篇数（published doc）
	TotalChars  int       // 文档正文总字数（rune）
	LastUpdated time.Time // 最近一次文档更新
}

var (
	docsStatsMu       sync.RWMutex
	docsStatsProvider func() *DocsStats
)

// RegisterDocsStats 由 docs 插件在 Register 阶段注入统计函数（供数即热，页面每次现取）。
func RegisterDocsStats(fn func() *DocsStats) {
	docsStatsMu.Lock()
	defer docsStatsMu.Unlock()
	docsStatsProvider = fn
}

// CurrentDocsStats 取当前统计；插件未启用返回 nil（调用方按零值渲染）。
func CurrentDocsStats() *DocsStats {
	docsStatsMu.RLock()
	fn := docsStatsProvider
	docsStatsMu.RUnlock()
	if fn == nil {
		return nil
	}
	return fn()
}
