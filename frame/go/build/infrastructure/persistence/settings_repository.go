package persistence

import (
	"database/sql"
	"fmt"

	"ven_hybird/build/domain/setting"
)

// SettingsRepository 是 setting.Repository 的 MySQL 实现。
type SettingsRepository struct {
	db *sql.DB
}

// NewSettingsRepository 构造设置仓储。
func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// 接口合规断言。
var _ setting.Repository = (*SettingsRepository)(nil)

// Get 读取键值，不存在返回空串（sql.ErrNoRows 归一为空值，调用方回退默认值）。
func (r *SettingsRepository) Get(key string) (string, error) {
	var value string
	err := r.db.QueryRow("SELECT value FROM settings WHERE `key` = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get setting %q: %w", key, err)
	}
	return value, nil
}

// Set 写入键值（upsert）。
func (r *SettingsRepository) Set(key, value string) error {
	_, err := r.db.Exec(
		"INSERT INTO settings (`key`, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = VALUES(value)",
		key, value,
	)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}
