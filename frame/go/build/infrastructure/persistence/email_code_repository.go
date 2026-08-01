package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"ven_hybird/build/domain/emailcode"
)

// EmailCodeRepository 是 emailcode.Repository 的 MySQL 实现。
type EmailCodeRepository struct {
	db *sql.DB
}

// NewEmailCodeRepository 构造验证码仓储。
func NewEmailCodeRepository(db *sql.DB) *EmailCodeRepository {
	return &EmailCodeRepository{db: db}
}

// Create 新建验证码（同邮箱先清旧码）。
func (r *EmailCodeRepository) Create(email, codeHash string, expiresAt time.Time) error {
	if _, err := r.db.Exec("DELETE FROM email_codes WHERE email = ?", email); err != nil {
		return fmt.Errorf("clear old codes of %q: %w", email, err)
	}
	if _, err := r.db.Exec(
		"INSERT INTO email_codes (email, code_hash, expires_at) VALUES (?, ?, ?)",
		email, codeHash, expiresAt,
	); err != nil {
		return fmt.Errorf("create email code for %q: %w", email, err)
	}
	return nil
}

// Latest 取邮箱最新一条验证码，不存在返回 nil, nil。
func (r *EmailCodeRepository) Latest(email string) (*emailcode.Entry, error) {
	e := &emailcode.Entry{}
	err := r.db.QueryRow(
		"SELECT id, email, code_hash, attempts, expires_at FROM email_codes WHERE email = ? ORDER BY id DESC LIMIT 1",
		email,
	).Scan(&e.ID, &e.Email, &e.CodeHash, &e.Attempts, &e.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("latest email code of %q: %w", email, err)
	}
	return e, nil
}

// IncrAttempts 尝试次数 +1。
func (r *EmailCodeRepository) IncrAttempts(id int64) error {
	_, err := r.db.Exec("UPDATE email_codes SET attempts = attempts + 1 WHERE id = ?", id)
	return err
}

// Delete 删除验证码。
func (r *EmailCodeRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM email_codes WHERE id = ?", id)
	return err
}
