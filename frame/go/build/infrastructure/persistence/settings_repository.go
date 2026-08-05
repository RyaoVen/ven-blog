package persistence

import (
	"database/sql"
	"fmt"

	"ven_hybird/build/domain/setting"
	"ven_hybird/build/infrastructure/cipher"
)

// SettingsRepository 是 setting.Repository 的 MySQL 实现。
type SettingsRepository struct {
	db  *sql.DB
	enc *cipher.Cipher // nil = 密钥未配置（敏感键明文存储，启动时警告）
}

// NewSettingsRepository 构造设置仓储；enc 可为 nil（BLOG_SECRET_KEY 未配置时回退明文）。
func NewSettingsRepository(db *sql.DB, enc *cipher.Cipher) *SettingsRepository {
	return &SettingsRepository{db: db, enc: enc}
}

// 接口合规断言。
var _ setting.Repository = (*SettingsRepository)(nil)

// sensitiveKeys 存库前加密的键（可逆对称加密——读回原文使用，不能单向哈希）。
// 密钥泄露面从"库文件"收窄到"库文件 + 部署环境变量"。
var sensitiveKeys = map[string]bool{
	setting.KeySMTPPass:  true,
	setting.KeyLLMAPIKey: true,
}

// Get 读取键值，不存在返回空串（sql.ErrNoRows 归一为空值，调用方回退默认值）。
// 敏感键为密文时自动解密；旧明文（无 enc: 前缀）原样返回，兼容迁移。
func (r *SettingsRepository) Get(key string) (string, error) {
	var value string
	err := r.db.QueryRow("SELECT value FROM settings WHERE `key` = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get setting %q: %w", key, err)
	}
	if sensitiveKeys[key] && r.enc != nil {
		value, err = r.enc.Decrypt(value)
		if err != nil {
			return "", fmt.Errorf("get setting %q: %w", key, err)
		}
	}
	return value, nil
}

// Set 写入键值（upsert）；敏感键自动加密后入库（配置密钥后旧明文随首次写入迁移为密文）。
func (r *SettingsRepository) Set(key, value string) error {
	if sensitiveKeys[key] && r.enc != nil {
		var err error
		value, err = r.enc.Encrypt(value)
		if err != nil {
			return fmt.Errorf("set setting %q: %w", key, err)
		}
	}
	_, err := r.db.Exec(
		"INSERT INTO settings (`key`, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = VALUES(value)",
		key, value,
	)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}
