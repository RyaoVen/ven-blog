// Package apikeyapp API 密钥用例服务：生成 / 列表 / 吊销 / 鉴权。
package apikeyapp

import (
	"log"
	"strconv"
	"strings"
	"time"

	"ven_hybird/build/domain/apikey"
)

// ValidationError 用例入参校验失败（接口层映射为 400）。
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// KeyView 脱敏视图（永不含明文；接口层直接下发）。
type KeyView struct {
	ID         string     `json:"id"` // 字符串下发（对齐 PostView 契约）
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"` // 如 ven_ab12（展示用）
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"` // null = 从未使用
	RevokedAt  *time.Time `json:"revokedAt"`  // null = 在用
}

// Service API 密钥用例服务。
type Service struct {
	repo apikey.Repository
}

// NewService 构造 API 密钥用例服务。
func NewService(repo apikey.Repository) *Service {
	return &Service{repo: repo}
}

// CreateKey 生成密钥：GenerateKey → HashKey → repo.Create，返回明文与脱敏视图。
// 明文仅此一次返回，调用方展示后必须丢弃；name 非法返回 *ValidationError。
func (s *Service) CreateKey(userID int64, name string) (raw string, view KeyView, err error) {
	if msg := apikey.ValidateName(name); msg != "" {
		return "", KeyView{}, &ValidationError{Message: msg}
	}
	raw, err = apikey.GenerateKey()
	if err != nil {
		return "", KeyView{}, err
	}
	k := &apikey.ApiKey{
		UserID:  userID,
		Name:    strings.TrimSpace(name),
		KeyHash: apikey.HashKey(raw),
		Prefix:  apikey.DisplayPrefix(raw),
	}
	if err := s.repo.Create(k); err != nil {
		return "", KeyView{}, err
	}
	return raw, toView(k), nil
}

// ListKeys 返回该用户全部密钥的脱敏视图（创建时间倒序，含已吊销），永不含明文。
func (s *Service) ListKeys(userID int64) ([]KeyView, error) {
	keys, err := s.repo.ListByUser(userID)
	if err != nil {
		return nil, err
	}
	views := make([]KeyView, 0, len(keys))
	for _, k := range keys {
		views = append(views, toView(k))
	}
	return views, nil
}

// Revoke 吊销本人密钥（吊销即终态，无恢复入口）；
// 不存在 / 已吊销 / 非本人返回 apikey.ErrNotFound。
func (s *Service) Revoke(userID, id int64) error {
	return s.repo.Revoke(userID, id)
}

// AuthenticateKey 用明文密钥换取用户身份：HashKey → FindByHash →
// 已吊销返回 apikey.ErrRevoked → UpdateLastUsedAt（失败仅记日志，不阻断鉴权）→ 返回 userID。
// 每次调用实时查库（无缓存），吊销即刻生效。
// 【调用方】Unit 2 的 /api/mcp 网关中间件（Bearer 解析在网关侧做，本单元只定签名，不实现网关）。
func (s *Service) AuthenticateKey(rawKey string) (userID int64, err error) {
	k, err := s.repo.FindByHash(apikey.HashKey(rawKey))
	if err != nil {
		return 0, err
	}
	if k.Revoked() {
		return 0, apikey.ErrRevoked
	}
	if err := s.repo.UpdateLastUsedAt(k.ID, time.Now()); err != nil {
		log.Printf("apikey: update last_used_at of key %d failed: %v", k.ID, err)
	}
	return k.UserID, nil
}

// toView 领域实体 → 脱敏视图（时间零值 → nil，对齐 JSON null 语义）。
func toView(k *apikey.ApiKey) KeyView {
	view := KeyView{
		ID:        strconv.FormatInt(k.ID, 10),
		Name:      k.Name,
		Prefix:    k.Prefix,
		CreatedAt: k.CreatedAt,
	}
	if !k.LastUsedAt.IsZero() {
		t := k.LastUsedAt
		view.LastUsedAt = &t
	}
	if !k.RevokedAt.IsZero() {
		t := k.RevokedAt
		view.RevokedAt = &t
	}
	return view
}
