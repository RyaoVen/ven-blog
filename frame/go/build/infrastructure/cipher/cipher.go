// Package cipher 敏感配置加密（AES-256-GCM）：smtp_pass / llm_api_key 存库前加密。
// 密钥来自环境变量 BLOG_SECRET_KEY（32 字节 hex）；未配置时 New 返回 ErrKeyNotConfigured，
// 调用方（settings 仓储）回退明文存储并在启动时打警告——渐进式安全，不破坏现有部署。
package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrKeyNotConfigured 密钥未配置（BLOG_SECRET_KEY 缺失或非 32 字节 hex）。
var ErrKeyNotConfigured = errors.New("cipher: BLOG_SECRET_KEY not configured (32-byte hex required)")

// Prefix 密文前缀（识别密文 vs 旧明文数据，兼容迁移）。
const Prefix = "enc:"

// Cipher AES-256-GCM 加解密器（密钥启动期解析一次，运行期只读，可并发使用）。
type Cipher struct {
	aead cipher.AEAD
}

// New 从 BLOG_SECRET_KEY 构造加解密器；未配置/格式错误返回 ErrKeyNotConfigured。
func New() (*Cipher, error) {
	raw := os.Getenv("BLOG_SECRET_KEY")
	if raw == "" {
		return nil, ErrKeyNotConfigured
	}
	key, err := hex.DecodeString(raw)
	if err != nil || len(key) != 32 {
		return nil, ErrKeyNotConfigured
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher: %w", err)
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt 加密明文 → Prefix + base64(nonce + ciphertext)（每次随机 nonce，同明文密文不同）。
func (c *Cipher) Encrypt(plain string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("cipher: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plain), nil)
	return Prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt 解密 Prefix 前缀密文；非前缀输入原样返回（旧明文数据兼容，配置密钥后首次写入自动迁移）。
func (c *Cipher) Decrypt(raw string) (string, error) {
	if !strings.HasPrefix(raw, Prefix) {
		return raw, nil
	}
	sealed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(raw, Prefix))
	if err != nil {
		return "", fmt.Errorf("cipher: decode: %w", err)
	}
	nonceSize := c.aead.NonceSize()
	if len(sealed) < nonceSize {
		return "", fmt.Errorf("cipher: short ciphertext")
	}
	plain, err := c.aead.Open(nil, sealed[:nonceSize], sealed[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("cipher: decrypt: %w", err)
	}
	return string(plain), nil
}
