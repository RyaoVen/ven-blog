// SeedUsers 测试：BLOG_AUTHOR_PASSWORD 强制配置（拒绝路径）、reader 密码 env 覆盖、非空表跳过。
// 用内存假仓储（实现 userSeeder，本包测试专用）替代 MySQL。
package persistence

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"ven_hybird/build/domain/user"
)

// fakeSeedRepo 实现 SeedUsers 的最小仓储依赖（Count/Create）。
type fakeSeedRepo struct {
	users map[string]*user.User
	err   error
}

func newFakeSeedRepo() *fakeSeedRepo {
	return &fakeSeedRepo{users: map[string]*user.User{}}
}

func (r *fakeSeedRepo) Count() (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	return len(r.users), nil
}

func (r *fakeSeedRepo) Create(u *user.User) error {
	if r.err != nil {
		return r.err
	}
	if _, ok := r.users[u.Username]; ok {
		return user.ErrUsernameTaken
	}
	cp := *u
	r.users[u.Username] = &cp
	return nil
}

// checkPassword 断言用户密码哈希与明文匹配。
func checkPassword(t *testing.T, u *user.User, password string) {
	t.Helper()
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		t.Fatalf("用户 %s 的密码哈希与 %q 不匹配: %v", u.Username, password, err)
	}
}

// TestSeedUsers_RequiresAuthorPassword 未配置 BLOG_AUTHOR_PASSWORD 拒绝启动且不写任何用户。
func TestSeedUsers_RequiresAuthorPassword(t *testing.T) {
	repo := newFakeSeedRepo()
	t.Setenv("BLOG_AUTHOR_PASSWORD", "")
	t.Setenv("BLOG_READER_PASSWORD", "")
	t.Setenv("BLOG_AUTHOR_NAME", "")
	err := SeedUsers(repo)
	if err == nil {
		t.Fatal("未配置 BLOG_AUTHOR_PASSWORD 应返回 error 拒绝启动")
	}
	if !strings.Contains(err.Error(), "BLOG_AUTHOR_PASSWORD") {
		t.Fatalf("错误信息应提示配置 BLOG_AUTHOR_PASSWORD，got: %v", err)
	}
	if len(repo.users) != 0 {
		t.Fatalf("拒绝路径不应写入任何用户，got %d", len(repo.users))
	}
}

// TestSeedUsers_CreatesAuthorAndReader 配置后写入 author（默认名）+ reader（默认密码）。
func TestSeedUsers_CreatesAuthorAndReader(t *testing.T) {
	repo := newFakeSeedRepo()
	t.Setenv("BLOG_AUTHOR_PASSWORD", "s3cret-author")
	t.Setenv("BLOG_READER_PASSWORD", "")
	t.Setenv("BLOG_AUTHOR_NAME", "")
	if err := SeedUsers(repo); err != nil {
		t.Fatalf("SeedUsers: %v", err)
	}
	author, ok := repo.users["author"]
	if !ok {
		t.Fatal("应创建 author 用户")
	}
	if author.Role != user.RoleAuthor {
		t.Fatalf("author 角色应为 author，got %v", author.Role)
	}
	checkPassword(t, author, "s3cret-author")
	reader, ok := repo.users["reader"]
	if !ok {
		t.Fatal("应创建 reader 用户")
	}
	if reader.Role != user.RoleReader {
		t.Fatalf("reader 角色应为 reader，got %v", reader.Role)
	}
	checkPassword(t, reader, "reader123")
}

// TestSeedUsers_EnvOverrides BLOG_AUTHOR_NAME/BLOG_READER_PASSWORD 可覆盖默认值。
func TestSeedUsers_EnvOverrides(t *testing.T) {
	repo := newFakeSeedRepo()
	t.Setenv("BLOG_AUTHOR_NAME", "me")
	t.Setenv("BLOG_AUTHOR_PASSWORD", "pw-author")
	t.Setenv("BLOG_READER_PASSWORD", "pw-reader")
	if err := SeedUsers(repo); err != nil {
		t.Fatalf("SeedUsers: %v", err)
	}
	if _, ok := repo.users["me"]; !ok {
		t.Fatal("author 用户名应取 BLOG_AUTHOR_NAME")
	}
	if _, ok := repo.users["author"]; ok {
		t.Fatal("默认用户名 author 不应出现")
	}
	checkPassword(t, repo.users["me"], "pw-author")
	checkPassword(t, repo.users["reader"], "pw-reader")
}

// TestSeedUsers_SkipsWhenUsersExist 非空表直接返回（不校验密码 env，老部署不受影响）。
func TestSeedUsers_SkipsWhenUsersExist(t *testing.T) {
	repo := newFakeSeedRepo()
	repo.users["someone"] = &user.User{Username: "someone", Role: user.RoleReader}
	t.Setenv("BLOG_AUTHOR_PASSWORD", "")
	if err := SeedUsers(repo); err != nil {
		t.Fatalf("非空表应直接返回 nil，got %v", err)
	}
	if len(repo.users) != 1 {
		t.Fatalf("不应新增种子用户，got %d", len(repo.users))
	}
}

// TestSeedUsers_PropagatesRepoErrors 仓储错误原样上抛。
func TestSeedUsers_PropagatesRepoErrors(t *testing.T) {
	repo := newFakeSeedRepo()
	repo.err = errors.New("db down")
	t.Setenv("BLOG_AUTHOR_PASSWORD", "pw")
	if err := SeedUsers(repo); !errors.Is(err, repo.err) {
		t.Fatalf("应原样返回仓储错误，got: %v", err)
	}
}
