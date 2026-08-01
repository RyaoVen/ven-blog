// Package settingsapp 站点设置用例服务：类型化访问键值配置（行格式编解码 + 内置默认值）。
package settingsapp

import (
	"strings"

	"ven_hybird/build/domain/setting"
)

// Skill 技能标签（level：deep/solid/know）。
type Skill struct {
	Name  string `json:"name"`
	Level string `json:"level"`
}

// FriendLink 友链。
type FriendLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Desc string `json:"desc"`
}

// Quote 收藏的句子。
type Quote struct {
	Text   string `json:"text"`
	Source string `json:"source"`
}

// Project 维护的项目。
type Project struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
	URL  string `json:"url"`
}

// Content 站点内容配置（设置页整体编辑/下发）。
type Content struct {
	Paragraphs []string     `json:"paragraphs"`
	Skills     []Skill      `json:"skills"`
	Friends    []FriendLink `json:"friends"`
	Quotes     []Quote      `json:"quotes"`
	Projects   []Project    `json:"projects"`
	GitHub     string       `json:"github"`
}

// Service 站点设置用例服务。
type Service struct {
	repo setting.Repository
}

// NewService 构造站点设置用例服务。
func NewService(repo setting.Repository) *Service {
	return &Service{repo: repo}
}

// Categories 文章分类列表（行格式存储；缺省回退内置默认）。
func (s *Service) Categories() ([]string, error) {
	raw, err := s.repo.Get(setting.KeyCategories)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		raw = strings.Join(defaultCategories, "\n")
	}
	return splitLines(raw), nil
}

// SetCategories 保存分类列表。
func (s *Service) SetCategories(categories []string) error {
	return s.repo.Set(setting.KeyCategories, strings.Join(categories, "\n"))
}

// Moderation 评论审核开关（开时新评论待审核）。
func (s *Service) Moderation() (bool, error) {
	raw, err := s.repo.Get(setting.KeyCommentModeration)
	if err != nil {
		return false, err
	}
	return raw == "on", nil
}

// SetModeration 设置评论审核开关。
func (s *Service) SetModeration(on bool) error {
	value := "off"
	if on {
		value = "on"
	}
	return s.repo.Set(setting.KeyCommentModeration, value)
}

// Content 读取站点内容配置（缺省项回退内置默认值）。
func (s *Service) Content() (*Content, error) {
	c := &Content{GitHub: defaultGitHub}
	var err error
	if c.Paragraphs, err = s.lines(setting.KeyIntroParagraphs, defaultParagraphs); err != nil {
		return nil, err
	}
	if c.Skills, err = s.skills(); err != nil {
		return nil, err
	}
	if c.Friends, err = s.friends(); err != nil {
		return nil, err
	}
	if c.Quotes, err = s.quotes(); err != nil {
		return nil, err
	}
	if c.Projects, err = s.projects(); err != nil {
		return nil, err
	}
	return c, nil
}

// SetContent 整体保存站点内容配置。
func (s *Service) SetContent(c *Content) error {
	if err := s.repo.Set(setting.KeyIntroParagraphs, strings.Join(nonEmptyLines(c.Paragraphs), "\n")); err != nil {
		return err
	}
	if err := s.repo.Set(setting.KeySkills, joinSkills(c.Skills)); err != nil {
		return err
	}
	if err := s.repo.Set(setting.KeyFriendLinks, joinTriples(c.Friends)); err != nil {
		return err
	}
	if err := s.repo.Set(setting.KeyQuotes, joinPairs(c.Quotes)); err != nil {
		return err
	}
	return s.repo.Set(setting.KeyProjects, joinProjects(c.Projects))
}

// lines 读取行格式配置，空值回退默认。
func (s *Service) lines(key string, fallback []string) ([]string, error) {
	raw, err := s.repo.Get(key)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		raw = strings.Join(fallback, "\n")
	}
	return splitLines(raw), nil
}

func (s *Service) skills() ([]Skill, error) {
	raw, err := s.repo.Get(setting.KeySkills)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return defaultSkills, nil
	}
	return parseSkills(raw), nil
}

func (s *Service) friends() ([]FriendLink, error) {
	raw, err := s.repo.Get(setting.KeyFriendLinks)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return defaultFriends, nil
	}
	return parseTriples(raw), nil
}

func (s *Service) quotes() ([]Quote, error) {
	raw, err := s.repo.Get(setting.KeyQuotes)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return defaultQuotes, nil
	}
	return parsePairs(raw), nil
}

func (s *Service) projects() ([]Project, error) {
	raw, err := s.repo.Get(setting.KeyProjects)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return defaultProjects, nil
	}
	return parseProjects(raw), nil
}

/* ===== 行格式编解码 ===== */

func splitLines(raw string) []string {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func nonEmptyLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseSkills(raw string) []Skill {
	out := make([]Skill, 0)
	for _, line := range splitLines(raw) {
		name, level, _ := strings.Cut(line, "|")
		out = append(out, Skill{Name: strings.TrimSpace(name), Level: strings.TrimSpace(level)})
	}
	return out
}

func joinSkills(skills []Skill) string {
	lines := make([]string, 0, len(skills))
	for _, s := range skills {
		if strings.TrimSpace(s.Name) != "" {
			lines = append(lines, strings.TrimSpace(s.Name)+"|"+strings.TrimSpace(s.Level))
		}
	}
	return strings.Join(lines, "\n")
}

func parseTriples(raw string) []FriendLink {
	out := make([]FriendLink, 0)
	for _, line := range splitLines(raw) {
		parts := strings.SplitN(line, "|", 3)
		item := FriendLink{}
		if len(parts) > 0 {
			item.Name = strings.TrimSpace(parts[0])
		}
		if len(parts) > 1 {
			item.URL = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 {
			item.Desc = strings.TrimSpace(parts[2])
		}
		out = append(out, item)
	}
	return out
}

func joinTriples(items []FriendLink) string {
	lines := make([]string, 0, len(items))
	for _, f := range items {
		if strings.TrimSpace(f.Name) != "" {
			lines = append(lines, strings.TrimSpace(f.Name)+"|"+strings.TrimSpace(f.URL)+"|"+strings.TrimSpace(f.Desc))
		}
	}
	return strings.Join(lines, "\n")
}

func parsePairs(raw string) []Quote {
	out := make([]Quote, 0)
	for _, line := range splitLines(raw) {
		text, source, _ := strings.Cut(line, "|")
		out = append(out, Quote{Text: strings.TrimSpace(text), Source: strings.TrimSpace(source)})
	}
	return out
}

func joinPairs(items []Quote) string {
	lines := make([]string, 0, len(items))
	for _, q := range items {
		if strings.TrimSpace(q.Text) != "" {
			lines = append(lines, strings.TrimSpace(q.Text)+"|"+strings.TrimSpace(q.Source))
		}
	}
	return strings.Join(lines, "\n")
}

func parseProjects(raw string) []Project {
	out := make([]Project, 0)
	for _, line := range splitLines(raw) {
		parts := strings.SplitN(line, "|", 3)
		item := Project{}
		if len(parts) > 0 {
			item.Name = strings.TrimSpace(parts[0])
		}
		if len(parts) > 1 {
			item.Desc = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 {
			item.URL = strings.TrimSpace(parts[2])
		}
		out = append(out, item)
	}
	return out
}

func joinProjects(items []Project) string {
	lines := make([]string, 0, len(items))
	for _, p := range items {
		if strings.TrimSpace(p.Name) != "" {
			lines = append(lines, strings.TrimSpace(p.Name)+"|"+strings.TrimSpace(p.Desc)+"|"+strings.TrimSpace(p.URL))
		}
	}
	return strings.Join(lines, "\n")
}
