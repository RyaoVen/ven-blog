package persistence

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ven_hybird/build/domain/visit"
)

// VisitRepository 是 visit.Repository 的 MySQL 实现。
type VisitRepository struct {
	db *sql.DB
}

// NewVisitRepository 构造访问统计仓储。
func NewVisitRepository(db *sql.DB) *VisitRepository {
	return &VisitRepository{db: db}
}

// Record 按 (date, path) 累计 +1：upsert，同日同路径重复调用幂等累加。
func (r *VisitRepository) Record(date time.Time, path string) error {
	_, err := r.db.Exec(
		"INSERT INTO visits (date, path, count) VALUES (?, ?, 1) ON DUPLICATE KEY UPDATE count = count + 1",
		date.Format("2006-01-02"), path,
	)
	return err
}

// Totals 全站访问总量与文章页（/posts/{id}）点击总量。
func (r *VisitRepository) Totals() (total int, postTotal int, err error) {
	if err := r.db.QueryRow("SELECT COALESCE(SUM(count), 0) FROM visits").Scan(&total); err != nil {
		return 0, 0, fmt.Errorf("visit totals: %w", err)
	}
	if err := r.db.QueryRow("SELECT COALESCE(SUM(count), 0) FROM visits WHERE path LIKE '/posts/%'").Scan(&postTotal); err != nil {
		return 0, 0, fmt.Errorf("visit post totals: %w", err)
	}
	return total, postTotal, nil
}

// Daily 近 days 天每日访问总量（GROUP BY date，Go 侧补零，日期升序）。
func (r *VisitRepository) Daily(days int) ([]visit.DailyCount, error) {
	if days <= 0 {
		days = 30
	}
	rows, err := r.db.Query(
		"SELECT date, count FROM visits WHERE date >= DATE_SUB(CURDATE(), INTERVAL ? DAY) ORDER BY date",
		days-1,
	)
	if err != nil {
		return nil, fmt.Errorf("visit daily: %w", err)
	}
	defer func() { _ = rows.Close() }()
	counts := make(map[string]int)
	for rows.Next() {
		var d time.Time
		var n int
		if err := rows.Scan(&d, &n); err != nil {
			return nil, fmt.Errorf("scan visit daily: %w", err)
		}
		counts[d.Format("2006-01-02")] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]visit.DailyCount, 0, days)
	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		out = append(out, visit.DailyCount{Date: date, Count: counts[date]})
	}
	return out, nil
}

// PostHits 各文章累计点击（path LIKE '/posts/%' 聚合，键为文章 ID）。
func (r *VisitRepository) PostHits() (map[int64]int, error) {
	rows, err := r.db.Query("SELECT path, count FROM visits WHERE path LIKE '/posts/%'")
	if err != nil {
		return nil, fmt.Errorf("visit post hits: %w", err)
	}
	defer func() { _ = rows.Close() }()
	hits := make(map[int64]int)
	for rows.Next() {
		var path string
		var n int
		if err := rows.Scan(&path, &n); err != nil {
			return nil, fmt.Errorf("scan visit post hit: %w", err)
		}
		// /posts/{id}：前缀后剩余部分即文章 ID，非数字（如子路径）跳过
		id, err := strconv.ParseInt(strings.TrimPrefix(path, "/posts/"), 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		hits[id] += n
	}
	return hits, rows.Err()
}
