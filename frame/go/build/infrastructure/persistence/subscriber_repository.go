package persistence

import (
	"database/sql"
	"fmt"

	"ven_hybird/build/domain/subscriber"
)

// SubscriberRepository 是 subscriber.Repository 的 MySQL 实现。
type SubscriberRepository struct {
	db *sql.DB
}

// NewSubscriberRepository 构造订阅者仓储。
func NewSubscriberRepository(db *sql.DB) *SubscriberRepository {
	return &SubscriberRepository{db: db}
}

// Create 记录订阅；邮箱唯一键冲突转 subscriber.ErrAlreadySubscribed。
func (r *SubscriberRepository) Create(s *subscriber.Subscriber) error {
	res, err := r.db.Exec("INSERT INTO subscribers (email) VALUES (?)", s.Email)
	if err != nil {
		if isDuplicateEntry(err) {
			return subscriber.ErrAlreadySubscribed
		}
		return fmt.Errorf("create subscriber %q: %w", s.Email, err)
	}
	s.ID, err = res.LastInsertId()
	return err
}

// Count 返回订阅者总数（后台统计）。
func (r *SubscriberRepository) Count() (int, error) {
	var n int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM subscribers").Scan(&n); err != nil {
		return 0, fmt.Errorf("count subscribers: %w", err)
	}
	return n, nil
}

// List 返回全部订阅者（按 ID 升序；订阅通知取收件人用）。
func (r *SubscriberRepository) List() ([]*subscriber.Subscriber, error) {
	rows, err := r.db.Query("SELECT id, email, created_at FROM subscribers ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("list subscribers: %w", err)
	}
	defer func() { _ = rows.Close() }()
	subs := make([]*subscriber.Subscriber, 0)
	for rows.Next() {
		var s subscriber.Subscriber
		if err := rows.Scan(&s.ID, &s.Email, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan subscriber: %w", err)
		}
		subs = append(subs, &s)
	}
	return subs, rows.Err()
}
