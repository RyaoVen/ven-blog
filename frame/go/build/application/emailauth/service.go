// Package emailauth 邮箱验证码用例服务：验证码签发（发邮件）与校验登录、@ 邮件通知。
package emailauth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"

	"ven_hybird/build/domain/emailcode"
	"ven_hybird/build/domain/user"
)

// 验证码参数：6 位数字、10 分钟有效、最多尝试 5 次。
const (
	codeTTL        = 10 * time.Minute
	maxAttempts    = 5
	codeNumberBase = 1000000
)

// Mailer 发送接口（与 infrastructure/mailer 同形，应用层不反向依赖具体实现）。
type Mailer interface {
	Send(to, subject, text string) error
}

// Service 邮箱验证码用例服务。
type Service struct {
	codes  emailcode.Repository
	users  user.Repository
	mailer Mailer
}

// NewService 构造邮箱验证码用例服务。
func NewService(codes emailcode.Repository, users user.Repository, mailer Mailer) *Service {
	return &Service{codes: codes, users: users, mailer: mailer}
}

// RequestCode 给邮箱签发验证码并发送（无论邮箱是否注册都返回成功——不泄露账号存在性）。
func (s *Service) RequestCode(email string) error {
	if msg := user.ValidateEmail(email); msg != "" {
		return &ValidationError{Message: msg}
	}
	code := generateCode()
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.codes.Create(email, string(hash), time.Now().Add(codeTTL)); err != nil {
		return err
	}
	subject := "ven-blog 登录验证码"
	text := fmt.Sprintf("你的登录验证码是：%s（10 分钟内有效，请勿泄露）。\n\n如果不是本人操作，请忽略本邮件。", code)
	// 发送失败不阻断流程（日志记录）；未配置 SMTP 时 mailer 降级日志输出验证码
	if err := s.mailer.Send(email, subject, text); err != nil {
		log.Printf("emailauth: send code to %s failed: %v", email, err)
	}
	return nil
}

// Verify 校验验证码并返回命中用户；验证码错误/过期/超尝试次数/邮箱未注册统一返回 ErrInvalidCode。
func (s *Service) Verify(email, code string) (*user.User, error) {
	entry, err := s.codes.Latest(email)
	if err != nil {
		return nil, err
	}
	if entry == nil || time.Now().After(entry.ExpiresAt) || entry.Attempts >= maxAttempts {
		return nil, ErrInvalidCode
	}
	if bcrypt.CompareHashAndPassword([]byte(entry.CodeHash), []byte(code)) != nil {
		_ = s.codes.IncrAttempts(entry.ID)
		return nil, ErrInvalidCode
	}
	u, err := s.users.FindByEmail(email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, ErrInvalidCode
		}
		return nil, err
	}
	_ = s.codes.Delete(entry.ID)
	return u, nil
}

// NotifyMentioned @ 邮件通知：replyTo 非空时解析用户并异步发送（不阻塞主流程，失败仅日志）。
// path 为原文站内路径（如 /posts/1 或 /moments），siteURL 为站点对外 URL。
func (s *Service) NotifyMentioned(replyTo, path, excerpt, siteURL string) {
	if replyTo == "" {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("emailauth: notify mentioned panic: %v", r)
			}
		}()
		target, err := s.users.FindByUsername(replyTo)
		if err != nil || target.Email == "" {
			return
		}
		subject := "有人在 ven-blog 的评论中提到了你"
		text := fmt.Sprintf("有人在评论中 @ 了你：\n\n「%s」\n\n查看原文：%s%s", excerpt, siteURL, path)
		if err := s.mailer.Send(target.Email, subject, text); err != nil {
			log.Printf("emailauth: notify %s failed: %v", target.Email, err)
		}
	}()
}

// generateCode 生成 6 位数字验证码（crypto/rand）。
func generateCode() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%codeNumberBase)
	}
	n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	if n < 0 {
		n = -n
	}
	return fmt.Sprintf("%06d", n%codeNumberBase)
}

// ErrInvalidCode 验证码错误/过期/超尝试/邮箱未注册（统一不泄露细节）。
var ErrInvalidCode = errors.New("invalid or expired code")

// ValidationError 用例入参校验失败（映射为 400）。
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }
