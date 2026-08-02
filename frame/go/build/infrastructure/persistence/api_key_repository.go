package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"ven_hybird/build/domain/apikey"
)

// ApiKeyRepository 是 apikey.Repository 的 MySQL 实现。
type ApiKeyRepository struct {
	db *sql.DB
}

// NewApiKeyRepository 构造 API 密钥仓储。
func NewApiKeyRepository(db *sql.DB) *ApiKeyRepository {
	return &ApiKeyRepository{db: db}
}

// apiKeyColumns 是 api_keys 表的查询列（FindByHash / ListByUser 共用）。
const apiKeyColumns = "id, user_id, name, key_hash, prefix, created_at, last_used_at, revoked_at"

// scanApiKey 扫描一行到实体（last_used_at / revoked_at NULL → 零值）。
func scanApiKey(scan func(dest ...any) error) (*apikey.ApiKey, error) {
	k := &apikey.ApiKey{}
	var lastUsedAt, revokedAt sql.NullTime
	if err := scan(&k.ID, &k.UserID, &k.Name, &k.KeyHash, &k.Prefix, &k.CreatedAt, &lastUsedAt, &revokedAt); err != nil {
		return nil, err
	}
	if lastUsedAt.Valid {
		k.LastUsedAt = lastUsedAt.Time
	}
	if revokedAt.Valid {
		k.RevokedAt = revokedAt.Time
	}
	return k, nil
}

// Create 新建密钥（user_id/name/key_hash/prefix），回填 ID 与 CreatedAt。
func (r *ApiKeyRepository) Create(k *apikey.ApiKey) error {
	res, err := r.db.Exec(
		"INSERT INTO api_keys (user_id, name, key_hash, prefix) VALUES (?, ?, ?, ?)",
		k.UserID, k.Name, k.KeyHash, k.Prefix,
	)
	if err != nil {
		return fmt.Errorf("create api key: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("api key last insert id: %w", err)
	}
	k.ID = id
	k.CreatedAt = time.Now()
	return nil
}

// FindByHash 按 hash 精确查找（唯一索引），不存在返回 apikey.ErrNotFound。
func (r *ApiKeyRepository) FindByHash(hash string) (*apikey.ApiKey, error) {
	k, err := scanApiKey(r.db.QueryRow(
		"SELECT "+apiKeyColumns+" FROM api_keys WHERE key_hash = ?",
		hash,
	).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apikey.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find api key by hash: %w", err)
	}
	return k, nil
}

// ListByUser 返回某用户全部密钥（创建时间倒序，含已吊销）。
func (r *ApiKeyRepository) ListByUser(userID int64) ([]*apikey.ApiKey, error) {
	rows, err := r.db.Query(
		"SELECT "+apiKeyColumns+" FROM api_keys WHERE user_id = ? ORDER BY id DESC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list api keys of user %d: %w", userID, err)
	}
	defer func() { _ = rows.Close() }()
	keys := make([]*apikey.ApiKey, 0)
	for rows.Next() {
		k, err := scanApiKey(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate api keys: %w", err)
	}
	return keys, nil
}

// Revoke 吊销（写 revoked_at）：仅限本人（user_id 匹配且未吊销）。
// 不存在 / 已吊销 / 非本人统一返回 apikey.ErrNotFound（不泄露 key 存在性）。
func (r *ApiKeyRepository) Revoke(userID, id int64) error {
	res, err := r.db.Exec(
		"UPDATE api_keys SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL",
		time.Now(), id, userID,
	)
	if err != nil {
		return fmt.Errorf("revoke api key %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("api key revoke rows affected: %w", err)
	}
	if affected == 0 {
		return apikey.ErrNotFound
	}
	return nil
}

// UpdateLastUsedAt 写最后使用时间（鉴权成功后调用）。
func (r *ApiKeyRepository) UpdateLastUsedAt(id int64, t time.Time) error {
	if _, err := r.db.Exec("UPDATE api_keys SET last_used_at = ? WHERE id = ?", t, id); err != nil {
		return fmt.Errorf("update api key last used at: %w", err)
	}
	return nil
}
