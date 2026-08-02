package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"ven_hybird/build/domain/moment"
)

// MomentRepository 是 moment.Repository 的 MySQL 实现。
type MomentRepository struct {
	db *sql.DB
}

// NewMomentRepository 构造动态仓储。
func NewMomentRepository(db *sql.DB) *MomentRepository {
	return &MomentRepository{db: db}
}

// 列表查询联表 users 取作者名。
const momentSelect = `SELECT m.id, m.author_id, u.username, m.content, m.pinned, m.created_at
FROM moments m JOIN users u ON u.id = m.author_id`

// scanMoment 从行扫描动态（列序与 momentSelect 一致）。
func scanMoment(row interface{ Scan(...any) error }) (*moment.Moment, error) {
	m := &moment.Moment{}
	err := row.Scan(&m.ID, &m.AuthorID, &m.AuthorName, &m.Content, &m.Pinned, &m.CreatedAt)
	return m, err
}

// List 返回最近动态（置顶优先，其内创建时间倒序），limit <= 0 表示全部。
func (r *MomentRepository) List(limit int) ([]*moment.Moment, error) {
	query := momentSelect + " ORDER BY m.pinned DESC, m.created_at DESC, m.id DESC"
	var args []any
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list moments: %w", err)
	}
	defer func() { _ = rows.Close() }()
	moments := make([]*moment.Moment, 0)
	for rows.Next() {
		m, err := scanMoment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan moment: %w", err)
		}
		moments = append(moments, m)
	}
	return moments, rows.Err()
}

// get 按 ID 取动态，不存在返回 moment.ErrNotFound。
func (r *MomentRepository) get(id int64) (*moment.Moment, error) {
	m, err := scanMoment(r.db.QueryRow(momentSelect+" WHERE m.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, moment.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get moment %d: %w", id, err)
	}
	return m, nil
}

// Create 新建动态并回填 ID 与时间戳。
func (r *MomentRepository) Create(m *moment.Moment) error {
	res, err := r.db.Exec("INSERT INTO moments (author_id, content) VALUES (?, ?)", m.AuthorID, m.Content)
	if err != nil {
		return fmt.Errorf("create moment: %w", err)
	}
	m.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	created, err := r.get(m.ID)
	if err != nil {
		return err
	}
	*m = *created
	return nil
}

// Delete 删除动态，不存在返回 moment.ErrNotFound。
func (r *MomentRepository) Delete(id int64) error {
	res, err := r.db.Exec("DELETE FROM moments WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete moment %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return moment.ErrNotFound
	}
	return nil
}

// SetPinned 设置动态置顶标记（仅改 pinned 列，不动时间戳），不存在返回 moment.ErrNotFound。
func (r *MomentRepository) SetPinned(id int64, pinned bool) error {
	res, err := r.db.Exec("UPDATE moments SET pinned = ? WHERE id = ?", pinned, id)
	if err != nil {
		return fmt.Errorf("set pinned moment %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return moment.ErrNotFound
	}
	return nil
}

// Count 返回动态总数（后台统计）。
func (r *MomentRepository) Count() (int, error) {
	var n int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM moments").Scan(&n); err != nil {
		return 0, fmt.Errorf("count moments: %w", err)
	}
	return n, nil
}

// DailyCounts 近 days 天每日发布数（GROUP BY DATE 聚合，日期 -> 篇数）。
func (r *MomentRepository) DailyCounts(days int) (map[string]int, error) {
	if days <= 0 {
		days = 365
	}
	rows, err := r.db.Query(
		"SELECT DATE(created_at) AS d, COUNT(*) FROM moments WHERE created_at >= DATE_SUB(CURDATE(), INTERVAL ? DAY) GROUP BY d",
		days-1,
	)
	if err != nil {
		return nil, fmt.Errorf("moment daily counts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	counts := make(map[string]int)
	for rows.Next() {
		var d time.Time
		var n int
		if err := rows.Scan(&d, &n); err != nil {
			return nil, fmt.Errorf("scan moment daily count: %w", err)
		}
		counts[d.Format("2006-01-02")] = n
	}
	return counts, rows.Err()
}
