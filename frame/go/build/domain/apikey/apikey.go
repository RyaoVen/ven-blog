// Package apikey API 密钥聚合：程序化鉴权凭据的实体与领域规则。
package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// 领域常量。
const (
	KeyPrefix        = "ven_" // 明文固定前缀（凭据体系标识，避免与 cookie/验证码混淆）
	RandomBytes      = 32     // 随机段字节数 → 明文总长 = 4 + 43 = 47（base64url 无填充）
	MaxNameLen       = 64     // name 上限（对齐 api_keys.name VARCHAR(64)）
	DisplayPrefixLen = 8      // 展示 prefix 截取长度（对齐 api_keys.prefix VARCHAR(16)）
)

// 领域错误（风格对齐 domain/user/repository.go）。
var (
	ErrNotFound = errors.New("api key not found")
	ErrRevoked  = errors.New("api key revoked")
)

// ApiKey API 密钥实体。
// 服务端唯一存储形态是 KeyHash；明文生命周期只存在于「创建时的一次返回」。
type ApiKey struct {
	ID         int64
	UserID     int64
	Name       string // 用途备注（如 "zcode-agent"）
	KeyHash    string // sha256(明文) 十六进制
	Prefix     string // 明文前 8 位，展示用
	CreatedAt  time.Time
	LastUsedAt time.Time // 零值 = 从未使用
	RevokedAt  time.Time // 零值 = 在用；非零 = 终态，不可恢复
}

// Revoked 是否已吊销（终态判定）。
func (k *ApiKey) Revoked() bool { return !k.RevokedAt.IsZero() }

// GenerateKey 生成明文密钥：ven_ + base64.RawURLEncoding(32 随机字节)，总长 47。
// 用 crypto/rand（不落日志、不落库，仅返回给创建方一次）。
func GenerateKey() (string, error) {
	buf := make([]byte, RandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return KeyPrefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashKey 计算明文密钥的 sha256 十六进制（存储形态）。
func HashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ValidateName 校验备注名：trim 后非空且 ≤ 64 字符；返回错误消息（空串 = 通过）。
func ValidateName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "name is required"
	}
	if utf8.RuneCountInString(name) > MaxNameLen {
		return "name too long (max 64)"
	}
	return ""
}

// DisplayPrefix 截取明文前 8 位作展示前缀（对齐 api_keys.prefix 列）。
func DisplayPrefix(raw string) string {
	if len(raw) < DisplayPrefixLen {
		return raw
	}
	return raw[:DisplayPrefixLen]
}
