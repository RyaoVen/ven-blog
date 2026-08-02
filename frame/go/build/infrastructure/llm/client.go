// Package llm 基础设施层：OpenAI 兼容 chat/completions 客户端，
// 实现 domain/moderation.Moderator（判定三态：approve/reject/pending）。
// 失败语义：一切无法判定的情况（网络/超时/非 2xx/解析失败/非法值）都返回 error，
// 由应用层决定重试与挂起——绝不把 error 映射为放行或驳回。
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ven_hybird/build/domain/moderation"
)

// Config 客户端配置（settings 键优先、env 兜底，见 docs/agent-design/unit-4-moderator-worker.md §9 配置总表）。
type Config struct {
	BaseURL string // OpenAI 兼容端点，默认 https://api.deepseek.com/v1
	APIKey  string // API key（为空时 Review 返回错误，worker 视为判定失败）
	Model   string // 模型名，默认 deepseek-chat
}

// 默认值（settings/env 均未配置时回退）。
const (
	defaultBaseURL = "https://api.deepseek.com/v1"
	defaultModel   = "deepseek-chat"
)

// NewClient 构造客户端；configFn 每次判定现取配置（设置页改动即时生效，无需重启）。
func NewClient(configFn func() (Config, error)) *Client {
	return &Client{configFn: configFn, http: &http.Client{Timeout: 30 * time.Second}}
}

// Client 实现 domain/moderation.Moderator。
type Client struct {
	configFn func() (Config, error)
	http     *http.Client // Timeout: 30s 硬超时兜底（同时尊重 ctx 取消）
}

// chatMessage 请求消息。
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest 请求体（temperature=0、JSON 输出、仅一条 user 消息）。
type chatRequest struct {
	Model          string            `json:"model"`
	Temperature    int               `json:"temperature"`
	ResponseFormat map[string]string `json:"response_format"`
	Messages       []chatMessage     `json:"messages"`
}

// chatResponse 期望响应：choices[0].message.content 为 JSON 字符串。
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Review 判定一条内容；所有无法判定的情况都返回 error。
func (c *Client) Review(ctx context.Context, req moderation.Request) (moderation.Verdict, error) {
	cfg, err := c.configFn()
	if err != nil {
		return moderation.Verdict{}, fmt.Errorf("llm: load config: %w", err)
	}
	if cfg.APIKey == "" {
		return moderation.Verdict{}, errors.New("llm: api key is not configured")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.Model == "" {
		cfg.Model = defaultModel
	}
	payload := chatRequest{
		Model:       cfg.Model,
		Temperature: 0,
		ResponseFormat: map[string]string{
			"type": "json_object",
		},
		Messages: []chatMessage{{Role: "user", Content: buildPrompt(req)}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return moderation.Verdict{}, fmt.Errorf("llm: marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(cfg.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return moderation.Verdict{}, fmt.Errorf("llm: build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return moderation.Verdict{}, fmt.Errorf("llm: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return moderation.Verdict{}, fmt.Errorf("llm: http status %d", resp.StatusCode)
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return moderation.Verdict{}, fmt.Errorf("llm: read body: %w", err)
	}
	var chat chatResponse
	if err := json.Unmarshal(respBody, &chat); err != nil {
		return moderation.Verdict{}, fmt.Errorf("llm: parse response: %w", err)
	}
	if len(chat.Choices) == 0 || strings.TrimSpace(chat.Choices[0].Message.Content) == "" {
		return moderation.Verdict{}, errors.New("llm: response missing choices[0].message.content")
	}
	return parseVerdict(chat.Choices[0].Message.Content)
}

// buildPrompt 拼装单条 user 消息（规则段 + 输入段 + 输出段全部内嵌，
// 不依赖 system role——对不支持/不擅长 system role 的 OpenAI 兼容端点兼容性最好）。
func buildPrompt(req moderation.Request) string {
	kind := "comment（评论，宿主为文章或动态）"
	if req.Host == moderation.HostGuestbook {
		kind = "guestbook（留言板留言）"
	}
	host := req.HostTitle
	if host == "" {
		host = "（未知）"
	}
	replyTo := "（无）"
	if req.ReplyTo != "" {
		replyTo = "@" + req.ReplyTo
	}
	return "你是内容审核助手。请判断下面这条用户内容是否违反站点的内容规范。\n\n" +
		"【审核规则】\n" +
		"判为 reject（违规）的情况——出现任意一条即可：\n" +
		"1. 垃圾广告/营销：售卖推广、代购、刷单、互粉互赞，或诱导点击外链（如\"加微信\"\"点击链接领红包\"\"扫码进群\"）。\n" +
		"2. 辱骂攻击：人身攻击、歧视、威胁、骚扰、贬低他人。\n" +
		"3. 引战挑衅：故意挑起对立、地域黑、无意义争吵、拉踩。\n" +
		"4. 敏感违规：政治敏感、违法、色情、赌博、暴力内容。\n" +
		"5. 空泛灌水：纯表情、无意义重复、凑字数刷存在感。\n\n" +
		"判为 approve（正常）的情况：与内容相关的正常交流、提问、指正、感谢、友好讨论，即使观点不同。\n\n" +
		"判为 pending（不确定）的情况：无法明确判断是否违规时。注意：宁可 pending 交由人工复核，也不要误判正常内容为违规；同样不应放过明显违规内容。\n\n" +
		"【输入内容】\n" +
		"- 内容类型：" + kind + "\n" +
		"- 宿主：" + host + "\n" +
		"- 回复对象（若有）：" + replyTo + "\n" +
		"- 内容正文：\n" + req.Content + "\n\n" +
		"【输出要求】\n" +
		"只输出一个 JSON 对象，不要输出任何其他文字、不要用 Markdown 代码块包裹。格式严格如下：\n" +
		"{\"verdict\": \"approve\" | \"reject\" | \"pending\", \"reason\": \"判定理由\"}\n" +
		"其中 verdict 只能是 approve、reject、pending 三者之一；reason 为字符串：reject 时必须说明违反的具体规则（如\"包含广告引流链接\"），approve/pending 可为空字符串。"
}

// parseVerdict 宽松解析模型输出：去空白 → 剥 Markdown 围栏 → 取首 { 到尾 } 的切片 → Unmarshal。
// 非法 verdict / reject 缺 reason 都视为无法判定 → error（绝不按非法值放行或驳回）。
func parseVerdict(raw string) (moderation.Verdict, error) {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimSpace(s[3:])
		s = strings.TrimPrefix(s, "json")
		s = strings.TrimSpace(strings.TrimSuffix(s, "```"))
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return moderation.Verdict{}, fmt.Errorf("llm: response has no JSON object: %q", raw)
	}
	var out struct {
		Verdict string `json:"verdict"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(s[start:end+1]), &out); err != nil {
		return moderation.Verdict{}, fmt.Errorf("llm: parse verdict json: %w", err)
	}
	switch out.Verdict {
	case moderation.ActionApprove, moderation.ActionReject, moderation.ActionPending:
	default:
		return moderation.Verdict{}, fmt.Errorf("llm: invalid verdict %q", out.Verdict)
	}
	if out.Verdict == moderation.ActionReject && strings.TrimSpace(out.Reason) == "" {
		return moderation.Verdict{}, errors.New("llm: reject verdict without reason")
	}
	return moderation.Verdict{Action: out.Verdict, Reason: out.Reason}, nil
}
