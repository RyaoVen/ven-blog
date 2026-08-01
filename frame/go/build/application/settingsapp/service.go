// Package settingsapp 站点设置用例服务：类型化访问键值配置（行格式编解码 + 内置默认值）。
package settingsapp

import (
	"strconv"
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

// SMTPConfig 读取 SMTP 配置（mailer 每次发送时现取）。
func (s *Service) SMTPConfig() (host, port, user, password, fromName string, err error) {
	if host, err = s.repo.Get(setting.KeySMTPHost); err != nil {
		return
	}
	if port, err = s.repo.Get(setting.KeySMTPPort); err != nil {
		return
	}
	if user, err = s.repo.Get(setting.KeySMTPUser); err != nil {
		return
	}
	if password, err = s.repo.Get(setting.KeySMTPPass); err != nil {
		return
	}
	fromName, err = s.repo.Get(setting.KeySMTPFromName)
	return
}

// SetSMTP 保存 SMTP 配置（password 为空串时保留原值——接口掩码场景）。
func (s *Service) SetSMTP(host, port, user, password, fromName string) error {
	pairs := map[string]string{
		setting.KeySMTPHost:     host,
		setting.KeySMTPPort:     port,
		setting.KeySMTPUser:     user,
		setting.KeySMTPFromName: fromName,
	}
	for k, v := range pairs {
		if err := s.repo.Set(k, v); err != nil {
			return err
		}
	}
	if password != "" {
		return s.repo.Set(setting.KeySMTPPass, password)
	}
	return nil
}

// AuthorEmail 作者个人邮箱（@ 通知收件地址）。
func (s *Service) AuthorEmail() (string, error) {
	return s.repo.Get(setting.KeyAuthorEmail)
}

// SetAuthorEmail 保存作者个人邮箱。
func (s *Service) SetAuthorEmail(email string) error {
	return s.repo.Set(setting.KeyAuthorEmail, email)
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

// SetParagraphs 保存作者介绍段落。
func (s *Service) SetParagraphs(paragraphs []string) error {
	return s.repo.Set(setting.KeyIntroParagraphs, strings.Join(nonEmptyLines(paragraphs), "\n"))
}

// SetSkills 保存技能标签。
func (s *Service) SetSkills(skills []Skill) error {
	return s.repo.Set(setting.KeySkills, joinSkills(skills))
}

// SetFriends 保存友链。
func (s *Service) SetFriends(friends []FriendLink) error {
	return s.repo.Set(setting.KeyFriendLinks, joinTriples(friends))
}

// SetQuotes 保存收藏的句子。
func (s *Service) SetQuotes(quotes []Quote) error {
	return s.repo.Set(setting.KeyQuotes, joinPairs(quotes))
}

// SetProjects 保存维护的项目。
func (s *Service) SetProjects(projects []Project) error {
	return s.repo.Set(setting.KeyProjects, joinProjects(projects))
}

// ShowcasePostsMax 展示柜文章位数（与作者主页裱框卡位一致）。
const ShowcasePostsMax = 4

// ShowcasePosts 展示柜文章 ID（有序；空表示未配置，页面回退最新文章）。
func (s *Service) ShowcasePosts() ([]int64, error) {
	raw, err := s.repo.Get(setting.KeyShowcasePosts)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, ShowcasePostsMax)
	for _, line := range splitLines(raw) {
		id, parseErr := strconv.ParseInt(line, 10, 64)
		if parseErr == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// SetShowcasePosts 保存展示柜文章 ID（超出位数的截断）。
func (s *Service) SetShowcasePosts(ids []int64) error {
	if len(ids) > ShowcasePostsMax {
		ids = ids[:ShowcasePostsMax]
	}
	lines := make([]string, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			lines = append(lines, strconv.FormatInt(id, 10))
		}
	}
	return s.repo.Set(setting.KeyShowcasePosts, strings.Join(lines, "\n"))
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
