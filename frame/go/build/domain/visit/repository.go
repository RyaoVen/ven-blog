package visit

import "time"

// Repository 访问统计仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// Record 按 (date, path) 累计 +1（同日内同路径重复调用幂等累加）。
	Record(date time.Time, path string) error
	// Totals 全站访问总量与文章页（path 形如 /posts/{id}）点击总量。
	Totals() (total int, postTotal int, err error)
	// Daily 近 days 天每日访问总量（日期升序，不足补零）。
	Daily(days int) ([]DailyCount, error)
	// PostHits 各文章累计点击（键为文章 ID，来自 /posts/{id} 路径聚合）。
	PostHits() (map[int64]int, error)
}
