package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	mysqldriver "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"

	"ven_hybird/build/domain/user"
)

// UserRepository 是 user.Repository 的 MySQL 实现。
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository 构造用户仓储。
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// 用户查询列序（与 scanUser 一致）。
const userSelect = "SELECT id, username, password_hash, role, bio, avatar_url, email, created_at FROM users"

// scanUser 从行扫描用户（列序与 userSelect 一致；email 为 NULLable）。
func scanUser(row interface{ Scan(...any) error }) (*user.User, error) {
	u := &user.User{}
	var email sql.NullString
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Bio, &u.AvatarURL, &email, &u.CreatedAt)
	u.Email = email.String
	return u, err
}

// FindByUsername 按用户名查找，不存在返回 user.ErrNotFound。
func (r *UserRepository) FindByUsername(username string) (*user.User, error) {
	u, err := scanUser(r.db.QueryRow(userSelect+" WHERE username = ?", username))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user %q: %w", username, err)
	}
	return u, nil
}

// FindByID 按 ID 查找，不存在返回 user.ErrNotFound。
func (r *UserRepository) FindByID(id int64) (*user.User, error) {
	u, err := scanUser(r.db.QueryRow(userSelect+" WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user %d: %w", id, err)
	}
	return u, nil
}

// Create 创建用户并回填 ID；用户名冲突返回 user.ErrUsernameTaken，邮箱冲突返回 user.ErrEmailTaken。
func (r *UserRepository) Create(u *user.User) error {
	res, err := r.db.Exec(
		"INSERT INTO users (username, password_hash, role, email) VALUES (?, ?, ?, NULLIF(?, ''))",
		u.Username, u.PasswordHash, u.Role.String(), u.Email,
	)
	if err != nil {
		if isDuplicateEntry(err) {
			if duplicateOnEmail(err) {
				return user.ErrEmailTaken
			}
			return user.ErrUsernameTaken
		}
		return fmt.Errorf("create user %q: %w", u.Username, err)
	}
	u.ID, err = res.LastInsertId()
	return err
}

// Count 返回用户总数。
func (r *UserRepository) Count() (int, error) {
	var n int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&n); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

// CountPosts 返回用户发布的文章数。
func (r *UserRepository) CountPosts(userID int64) (int, error) {
	var n int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM posts WHERE author_id = ?", userID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count posts of user %d: %w", userID, err)
	}
	return n, nil
}

// CountComments 返回用户发表的评论数。
func (r *UserRepository) CountComments(userID int64) (int, error) {
	var n int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM comments WHERE user_id = ?", userID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count comments of user %d: %w", userID, err)
	}
	return n, nil
}

// AuthorUsernameFromEnv 返回种子 author 用户名（BLOG_AUTHOR_NAME，默认 author）。
// 组装根用它定位"当前唯一可发文账号"（框架会话尚无用户身份，见 register.go 注释）。
func AuthorUsernameFromEnv() string {
	if name := os.Getenv("BLOG_AUTHOR_NAME"); name != "" {
		return name
	}
	return "author"
}

// SeedUsers 用户表为空时写入种子账号：
// author（BLOG_AUTHOR_NAME/BLOG_AUTHOR_PASSWORD 可覆盖，默认 author/author123）；
// reader（reader/reader123，用于验证 403）。密码 bcrypt 哈希。
func SeedUsers(repo *UserRepository) error {
	n, err := repo.Count()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	authorPassword := os.Getenv("BLOG_AUTHOR_PASSWORD")
	if authorPassword == "" {
		authorPassword = "author123"
	}
	seeds := []struct {
		username, password string
		role               user.Role
	}{
		{AuthorUsernameFromEnv(), authorPassword, user.RoleAuthor},
		{"reader", "reader123", user.RoleReader},
	}
	for _, s := range seeds {
		hash, err := bcrypt.GenerateFromPassword([]byte(s.password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := repo.Create(&user.User{Username: s.username, PasswordHash: string(hash), Role: s.role}); err != nil {
			return fmt.Errorf("seed user %q: %w", s.username, err)
		}
	}
	return nil
}

// isDuplicateEntry 判定 MySQL 1062 唯一键冲突。
func isDuplicateEntry(err error) bool {
	var mysqlErr *mysqldriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// UpdatePassword 更新密码哈希。
func (r *UserRepository) UpdatePassword(userID int64, passwordHash string) error {
	_, err := r.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", passwordHash, userID)
	if err != nil {
		return fmt.Errorf("update password of user %d: %w", userID, err)
	}
	return nil
}

// UpdateProfile 更新简介与头像。
func (r *UserRepository) UpdateProfile(userID int64, bio, avatarURL string) error {
	_, err := r.db.Exec("UPDATE users SET bio = ?, avatar_url = ? WHERE id = ?", bio, avatarURL, userID)
	if err != nil {
		return fmt.Errorf("update profile of user %d: %w", userID, err)
	}
	return nil
}

// FindByEmail 按邮箱查找（验证码登录），不存在返回 user.ErrNotFound。
func (r *UserRepository) FindByEmail(email string) (*user.User, error) {
	u, err := scanUser(r.db.QueryRow(userSelect+" WHERE email = ?", email))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email %q: %w", email, err)
	}
	return u, nil
}

// UpdateEmail 绑定/修改邮箱；唯一键冲突返回 user.ErrEmailTaken。
func (r *UserRepository) UpdateEmail(userID int64, email string) error {
	_, err := r.db.Exec("UPDATE users SET email = ? WHERE id = ?", email, userID)
	if err != nil {
		if isDuplicateEntry(err) {
			return user.ErrEmailTaken
		}
		return fmt.Errorf("update email of user %d: %w", userID, err)
	}
	return nil
}

// duplicateOnEmail 判定 1062 冲突是否落在 email 唯一键上（错误消息含键名）。
func duplicateOnEmail(err error) bool {
	var mysqlErr *mysqldriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 &&
		(strings.Contains(mysqlErr.Message, "email") || strings.Contains(mysqlErr.Message, "uk_users_email"))
}
