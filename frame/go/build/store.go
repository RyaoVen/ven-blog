// 博客的内存存储：文章表与用户表。
// 仅用于验证全链路，进程重启即丢；后续单元替换为真实存储。
package build

import (
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Post 是一篇博客文章。
type Post struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// User 是内存用户表中的账号（明文口令，仅开发期种子数据用）。
type User struct {
	Username string
	Password string
	Role     string
}

// blogStore 是博客业务共享的内存存储，读写由互斥锁保护。
type blogStore struct {
	mu     sync.RWMutex
	posts  map[string]*Post
	nextID int
	users  map[string]User // username → User
}

// newBlogStore 构造存储并写入种子用户：
// author 账号（BLOG_AUTHOR_NAME/BLOG_AUTHOR_PASSWORD 可覆盖，默认 author/author123）；
// reader 账号（reader/reader123，用于验证已登录但角色不足的 403 行为）。
func newBlogStore() *blogStore {
	authorName := os.Getenv("BLOG_AUTHOR_NAME")
	if authorName == "" {
		authorName = "author"
	}
	authorPassword := os.Getenv("BLOG_AUTHOR_PASSWORD")
	if authorPassword == "" {
		authorPassword = "author123"
	}
	return &blogStore{
		posts:  make(map[string]*Post),
		nextID: 1,
		users: map[string]User{
			authorName: {Username: authorName, Password: authorPassword, Role: "author"},
			"reader":   {Username: "reader", Password: "reader123", Role: "reader"},
		},
	}
}

// authenticate 校验用户名口令，通过返回用户。
func (s *blogStore) authenticate(username, password string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[username]
	if !ok || u.Password != password {
		return User{}, false
	}
	return u, true
}

// listPosts 返回全部文章，按创建时间倒序。
func (s *blogStore) listPosts() []*Post {
	s.mu.RLock()
	defer s.mu.RUnlock()
	posts := make([]*Post, 0, len(s.posts))
	for _, p := range s.posts {
		posts = append(posts, p)
	}
	sort.Slice(posts, func(i, j int) bool { return posts[i].CreatedAt.After(posts[j].CreatedAt) })
	return posts
}

// getPost 按 ID 取文章。
func (s *blogStore) getPost(id string) (*Post, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.posts[id]
	return p, ok
}

// createPost 新建文章并分配自增 ID。
func (s *blogStore) createPost(title, content string) *Post {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	p := &Post{
		ID:        strconv.Itoa(s.nextID),
		Title:     title,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.nextID++
	s.posts[p.ID] = p
	return p
}

// updatePost 更新标题与正文，文章不存在返回 false。
func (s *blogStore) updatePost(id, title, content string) (*Post, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.posts[id]
	if !ok {
		return nil, false
	}
	p.Title = title
	p.Content = content
	p.UpdatedAt = time.Now()
	return p, true
}

// deletePost 删除文章，文章不存在返回 false。
func (s *blogStore) deletePost(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.posts[id]; !ok {
		return false
	}
	delete(s.posts, id)
	return true
}
