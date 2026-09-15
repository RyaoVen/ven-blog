package docs

import (
	"strings"
)

// Frontmatter 简易 frontmatter 元数据（不引 YAML 库：每行 key: value，tags 逗号分隔）。
type Frontmatter struct {
	Title   string
	Summary string
	Tags    []string
	Order   *int
	Status  Status
}

// splitFrontmatter 拆分 "---\n...\n---\n" 头与正文。
// 无 frontmatter 头返回 (nil, 原文)。
func splitFrontmatter(source string) (map[string]string, string) {
	normalized := strings.ReplaceAll(source, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return nil, source
	}
	end := strings.Index(normalized[4:], "\n---")
	if end < 0 {
		return nil, source
	}
	header := normalized[4 : 4+end]
	rest := normalized[4+end:]
	rest = strings.TrimPrefix(rest, "\n---")
	// 去掉结束行剩余部分（到行尾）。
	if idx := strings.Index(rest, "\n"); idx >= 0 {
		rest = rest[idx+1:]
	} else {
		rest = ""
	}
	rest = strings.TrimLeft(rest, "\n") // 去掉 frontmatter 后的空行（往返归一）
	meta := make(map[string]string)
	for _, line := range strings.Split(header, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		colon := strings.Index(line, ":")
		if colon <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:colon])
		value := strings.TrimSpace(line[colon+1:])
		value = strings.Trim(value, `"'`)
		meta[strings.ToLower(key)] = value
	}
	return meta, rest
}

// parseFrontmatter 解析元数据与正文。
func parseFrontmatter(source string) (Frontmatter, string, error) {
	metaMap, body := splitFrontmatter(source)
	fm := Frontmatter{}
	if metaMap == nil {
		return fm, source, nil
	}
	fm.Title = metaMap["title"]
	fm.Summary = metaMap["summary"]
	if raw, ok := metaMap["tags"]; ok && raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				fm.Tags = append(fm.Tags, t)
			}
		}
	}
	if raw, ok := metaMap["order"]; ok && raw != "" {
		n := 0
		neg := false
		s := raw
		if strings.HasPrefix(s, "-") {
			neg = true
			s = s[1:]
		}
		for _, r := range s {
			if r < '0' || r > '9' {
				n = 0
				break
			}
			n = n*10 + int(r-'0')
		}
		if neg {
			n = -n
		}
		fm.Order = &n
	}
	if raw, ok := metaMap["status"]; ok && raw != "" {
		fm.Status = Status(raw)
	}
	return fm, body, nil
}

// buildFrontmatter 从 doc 还原 markdown（export 用；往返一致：import(export(x)) ≈ x）。
func buildFrontmatter(doc *Doc, withBody bool) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("title: " + doc.Title + "\n")
	if doc.Summary != "" {
		b.WriteString("summary: " + doc.Summary + "\n")
	}
	if len(doc.Tags) > 0 {
		b.WriteString("tags: " + strings.Join(doc.Tags, ", ") + "\n")
	}
	b.WriteString("order: " + itoa(doc.SortOrder) + "\n")
	b.WriteString("status: " + string(doc.Status) + "\n")
	b.WriteString("---\n\n")
	if withBody {
		b.WriteString(doc.Content)
	}
	return b.String()
}

// itoa 整数转字符串（避免测试外引 strconv——直接用标准库亦可，保持一处依赖）。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}
