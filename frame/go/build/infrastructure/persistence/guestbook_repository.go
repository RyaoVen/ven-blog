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

// List 返回留言（创建时间倒序，联表取用户名）。
func (r *GuestbookRepository) List(limit int) ([]*guestbook.Entry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(
		`SELECT g.id, g.user_id, u.username, g.content, g.created_at
		FROM guestbook g JOIN users u ON u.id = g.user_id
		ORDER BY g.created_at DESC, g.id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list guestbook: %w", err)
	}
	defer func() { _ = rows.Close() }()
	entries := make([]*guestbook.Entry, 0)
	for rows.Next() {
		e := &guestbook.Entry{}
		if err := rows.Scan(&e.ID, &e.UserID, &e.Username, &e.Content, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan guestbook entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Get 按 ID 取留言，不存在返回 guestbook.ErrNotFound。
func (r *GuestbookRepository) Get(id int64) (*guestbook.Entry, error) {
	e := &guestbook.Entry{}
	err := r.db.QueryRow(
		`SELECT g.id, g.user_id, u.username, g.content, g.created_at
		FROM guestbook g JOIN users u ON u.id = g.user_id WHERE g.id = ?`,
		id,
	).Scan(&e.ID, &e.UserID, &e.Username, &e.Content, &e.CreatedAt)
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
	res, err := r.db.Exec("INSERT INTO guestbook (user_id, content) VALUES (?, ?)", e.UserID, e.Content)
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
