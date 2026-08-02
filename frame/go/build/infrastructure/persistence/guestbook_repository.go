package persistence

import (
	"database/sql"
	"errors"
	"fmt"

	"ven_hybird/build/domain/guestbook"
)

// GuestbookRepository 是 guestbook.Repository 的 MySQL 实现。
type GuestbookRepository struct {
	db *sql.DB
}

// NewGuestbookRepository 构造留言板仓储。
func NewGuestbookRepository(db *sql.DB) *GuestbookRepository {
	return &GuestbookRepository{db: db}
}

// guestbookSelect 留言查询列。
const guestbookSelect = `SELECT g.id, g.user_id, u.username, g.content, g.status, g.rejected_reason, g.created_at
FROM guestbook g JOIN users u ON u.id = g.user_id`

// scanEntry 从行扫描留言。
func scanEntry(row interface{ Scan(...any) error }) (*guestbook.Entry, error) {
	e := &guestbook.Entry{}
	err := row.Scan(&e.ID, &e.UserID, &e.Username, &e.Content, &e.Status, &e.RejectedReason, &e.CreatedAt)
	return e, err
}

// List 返回公开留言（仅 approved，创建时间倒序，联表取用户名）。
func (r *GuestbookRepository) List(limit int) ([]*guestbook.Entry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(
		guestbookSelect+" WHERE g.status = ? ORDER BY g.created_at DESC, g.id DESC LIMIT ?",
		guestbook.StatusApproved, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list guestbook: %w", err)
	}
	defer func() { _ = rows.Close() }()
	entries := make([]*guestbook.Entry, 0)
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan guestbook entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// listWhere 按状态查询留言（asc 为 true 时创建时间正序，先审先到）。
func (r *GuestbookRepository) listWhere(status string, asc bool, limit int) ([]*guestbook.Entry, error) {
	order := "DESC"
	if asc {
		order = "ASC"
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(
		guestbookSelect+" WHERE g.status = ? ORDER BY g.created_at "+order+", g.id "+order+" LIMIT ?",
		status, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list guestbook %s: %w", status, err)
	}
	defer func() { _ = rows.Close() }()
	entries := make([]*guestbook.Entry, 0)
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan guestbook entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ListAll 返回全量留言（含全部状态，创建时间倒序，后台管理用）。
func (r *GuestbookRepository) ListAll(limit int) ([]*guestbook.Entry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(
		guestbookSelect+" ORDER BY g.created_at DESC, g.id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list all guestbook: %w", err)
	}
	defer func() { _ = rows.Close() }()
	entries := make([]*guestbook.Entry, 0)
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan guestbook entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ListPending 返回待审核留言（创建时间正序）。
func (r *GuestbookRepository) ListPending() ([]*guestbook.Entry, error) {
	return r.listWhere(guestbook.StatusPending, true, 0)
}

// ListRejected 返回被驳回留言（创建时间正序）。
func (r *GuestbookRepository) ListRejected() ([]*guestbook.Entry, error) {
	return r.listWhere(guestbook.StatusRejected, true, 0)
}

// Get 按 ID 取留言，不存在返回 guestbook.ErrNotFound。
func (r *GuestbookRepository) Get(id int64) (*guestbook.Entry, error) {
	e, err := scanEntry(r.db.QueryRow(guestbookSelect+" WHERE g.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, guestbook.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get guestbook entry %d: %w", id, err)
	}
	return e, nil
}

// Create 创建留言并回填 ID 与时间戳。
func (r *GuestbookRepository) Create(e *guestbook.Entry) error {
	res, err := r.db.Exec("INSERT INTO guestbook (user_id, content, status) VALUES (?, ?, ?)", e.UserID, e.Content, e.Status)
	if err != nil {
		return fmt.Errorf("create guestbook entry: %w", err)
	}
	e.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	created, err := r.Get(e.ID)
	if err != nil {
		return err
	}
	*e = *created
	return nil
}

// Delete 删除留言，不存在返回 guestbook.ErrNotFound。
func (r *GuestbookRepository) Delete(id int64) error {
	res, err := r.db.Exec("DELETE FROM guestbook WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete guestbook entry %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return guestbook.ErrNotFound
	}
	return nil
}

// SetStatus 更新留言状态（审核通过/打回），同时清空驳回原因。
func (r *GuestbookRepository) SetStatus(id int64, status string) error {
	if _, err := r.db.Exec("UPDATE guestbook SET status = ?, rejected_reason = '' WHERE id = ?", status, id); err != nil {
		return fmt.Errorf("set status of guestbook entry %d: %w", id, err)
	}
	return nil
}

// SetRejected 驳回留言并记录驳回原因。
func (r *GuestbookRepository) SetRejected(id int64, reason string) error {
	if _, err := r.db.Exec("UPDATE guestbook SET status = ?, rejected_reason = ? WHERE id = ?", guestbook.StatusRejected, reason, id); err != nil {
		return fmt.Errorf("reject guestbook entry %d: %w", id, err)
	}
	return nil
}
