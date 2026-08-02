// Package visit 访问统计聚合：逐日按路径累计的页面浏览量（PV）。
package visit

import "time"

// Visit 一条 (date, path) 维度的访问累计。
type Visit struct {
	Date  time.Time
	Path  string
	Count int
}

// DailyCount 某日全站访问总量。
type DailyCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}
