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
