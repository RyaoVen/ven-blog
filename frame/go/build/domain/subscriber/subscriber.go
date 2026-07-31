// Package subscriber 订阅者聚合：邮箱订阅记录。
package subscriber

import (
	"regexp"
	"time"
)

// Subscriber 邮箱订阅者。
type Subscriber struct {
	ID        int64
	Email     string
	CreatedAt time.Time
}

// emailPattern 实用级邮箱格式校验（非 RFC 完备，够用）。
var emailPattern = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)

// ValidateEmail 校验邮箱格式，返回错误消息（空串表示通过）。
func ValidateEmail(email string) string {
	if len(email) > 254 || !emailPattern.MatchString(email) {
		return "invalid email"
	}
	return ""
}
