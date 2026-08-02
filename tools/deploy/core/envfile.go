// Package core 提供部署工具的纯逻辑：env 解析、配置、环境检测、构建编排、进程管理、日志 tail。
package core

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// EntryKind 区分 .env.local 中的条目类型。
type EntryKind int

const (
	EntryKey EntryKind = iota
	EntryComment
	EntryBlank
)

// Entry 是 .env.local 中的一行（KEY=VALUE / 注释 / 空行）。
type Entry struct {
	Kind    EntryKind
	Key     string // Kind == EntryKey 时有效
	Value   string // 已剥离引号的值
	Comment string // Kind == EntryComment 时有效：整行原文（含 #）
}

// EnvFile 是 .env.local 的有序解析结果（保留注释与空行，便于无损回写）。
type EnvFile struct {
	Entries []Entry
}

var keyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidKey 校验环境变量键名（字母/数字/下划线，不能数字开头）。
func ValidKey(k string) bool { return keyRe.MatchString(k) }

// Parse 解析 .env.local 内容：KEY=VALUE、# 注释、空行。
// 值两侧空白被修剪；首尾成对的单/双引号被剥离。格式错误返回带行号的错误。
func Parse(data []byte) (*EnvFile, error) {
	f := &EnvFile{}
	lines := strings.Split(string(data), "\n")
	// 去掉结尾换行产生的空段（文件以 \n 结尾时 Split 会多出一个空元素）
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for i, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			f.Entries = append(f.Entries, Entry{Kind: EntryBlank})
		case strings.HasPrefix(trimmed, "#"):
			f.Entries = append(f.Entries, Entry{Kind: EntryComment, Comment: line})
		default:
			eq := strings.IndexByte(line, '=')
			if eq < 0 {
				return nil, fmt.Errorf("第 %d 行不是合法的 KEY=VALUE：%q", i+1, line)
			}
			key := strings.TrimSpace(line[:eq])
			if !ValidKey(key) {
				return nil, fmt.Errorf("第 %d 行键名非法：%q", i+1, key)
			}
			val := stripQuotes(strings.TrimSpace(line[eq+1:]))
			f.Entries = append(f.Entries, Entry{Kind: EntryKey, Key: key, Value: val})
		}
	}
	return f, nil
}

// stripQuotes 剥离首尾成对的单/双引号（双引号内 \" 与 \\ 反转义）。
func stripQuotes(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		inner := s[1 : len(s)-1]
		if s[0] == '"' {
			inner = strings.ReplaceAll(inner, `\\`, `\`)
			inner = strings.ReplaceAll(inner, `\"`, `"`)
		}
		return inner
	}
	return s
}

// Get 返回键值，不存在时 ok=false。
func (f *EnvFile) Get(key string) (string, bool) {
	for _, e := range f.Entries {
		if e.Kind == EntryKey && e.Key == key {
			return e.Value, true
		}
	}
	return "", false
}

// Set 追加或覆盖键：覆盖首处出现的同名键（保留原位置与注释），否则追加到末尾。
func (f *EnvFile) Set(key, value string) {
	for i := range f.Entries {
		if f.Entries[i].Kind == EntryKey && f.Entries[i].Key == key {
			f.Entries[i].Value = value
			return
		}
	}
	f.Entries = append(f.Entries, Entry{Kind: EntryKey, Key: key, Value: value})
}

// Keys 返回全部键（出现顺序）。
func (f *EnvFile) Keys() []string {
	var out []string
	for _, e := range f.Entries {
		if e.Kind == EntryKey {
			out = append(out, e.Key)
		}
	}
	return out
}

// AddComment 追加注释行（text 需自带 #）。
func (f *EnvFile) AddComment(text string) {
	f.Entries = append(f.Entries, Entry{Kind: EntryComment, Comment: text})
}

// AddBlank 追加空行。
func (f *EnvFile) AddBlank() {
	f.Entries = append(f.Entries, Entry{Kind: EntryBlank})
}

// Marshal 序列化回 .env.local 文本；值含空白/#/引号/= 时以双引号包裹。
// 注释与空行原样保留。
func (f *EnvFile) Marshal() []byte {
	var b strings.Builder
	for _, e := range f.Entries {
		switch e.Kind {
		case EntryBlank:
			b.WriteByte('\n')
		case EntryComment:
			b.WriteString(e.Comment)
			b.WriteByte('\n')
		case EntryKey:
			b.WriteString(e.Key)
			b.WriteByte('=')
			b.WriteString(quoteIfNeeded(e.Value))
			b.WriteByte('\n')
		}
	}
	return []byte(b.String())
}

// quoteIfNeeded 值含空白/#/引号/= 时用双引号包裹（兼容 bash source 与本解析器）。
func quoteIfNeeded(v string) string {
	if v == "" || !strings.ContainsAny(v, " \t#\"'=") {
		return v
	}
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return `"` + v + `"`
}

// LoadEnvFile 读取 .env.local；文件不存在时返回空文件（不报错）。
func LoadEnvFile(path string) (*EnvFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &EnvFile{}, nil
		}
		return nil, err
	}
	return Parse(data)
}

// Save 将内容写回 path。
func (f *EnvFile) Save(path string) error {
	return os.WriteFile(path, f.Marshal(), 0o644)
}
