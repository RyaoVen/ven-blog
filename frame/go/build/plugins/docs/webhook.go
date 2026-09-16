package docs

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"ven_hybird/build/plugin"
)

// WebhookEvent 出站事件（unit-7 §5）。
type WebhookEvent struct {
	Event string `json:"event"` // doc.created / doc.updated / doc.deleted / doc.moved
	Path  string `json:"path"`
	At    string `json:"at"`
}

// WebhookDispatcher 出站 webhook 投递器（Startable/Stoppable）：
// 写操作成功后异步入队，worker 独立 goroutine 投递；失败退避重试 3 次，最终失败仅记日志。
// 配置：settings plugin.docs.webhook_url / plugin.docs.webhook_secret——
// secret 属敏感键，经 persistence sensitiveKeys AES-GCM 加密落盘（未配置 BLOG_SECRET_KEY 时回退明文并启动警告，与主业务同策略）；
// 投递时现读配置：保存后即时生效，无需重启。
type WebhookDispatcher struct {
	store   plugin.SettingsStore
	client  *http.Client
	ch      chan WebhookEvent
	done    chan struct{}
	wg      sync.WaitGroup
	mu      sync.Mutex
	started bool
}

// NewWebhookDispatcher 构造投递器（纯构造；Start/Stop 由内核生命周期驱动）。
func NewWebhookDispatcher(store plugin.SettingsStore) *WebhookDispatcher {
	return &WebhookDispatcher{
		store:  store,
		client: &http.Client{Timeout: 5 * time.Second},
		ch:     make(chan WebhookEvent, 64),
		done:   make(chan struct{}),
	}
}

// Settings 配置键（secret 经 persistence sensitiveKeys 加密落盘）。
const (
	SettingWebhookURL    = "plugin.docs.webhook_url"
	SettingWebhookSecret = "plugin.docs.webhook_secret"
)

// Start 实现 plugin.Startable：启动 worker（投递时现读配置，保存后即时生效）。
func (w *WebhookDispatcher) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.started {
		return nil
	}
	w.started = true
	w.wg.Add(1)
	go w.loop()
	if url := w.config(); url != "" {
		log.Printf("docs: webhook 投递器已启动 → %s", url)
	} else {
		log.Printf("docs: webhook 投递器已启动（未配置 %s，事件将丢弃直至配置保存）", SettingWebhookURL)
	}
	return nil
}

// Stop 实现 plugin.Stoppable：关队列等 worker 退出（评审 P0 项：必须释放资源）。
func (w *WebhookDispatcher) Stop() error {
	w.mu.Lock()
	if w.started {
		close(w.done)
		w.started = false
	}
	w.mu.Unlock()
	w.wg.Wait()
	return nil
}

// Emit 异步入队事件（非阻塞；队列满丢弃并记日志——不阻塞写响应）。
func (w *WebhookDispatcher) Emit(event, path string) {
	w.mu.Lock()
	started := w.started
	w.mu.Unlock()
	if !started {
		return
	}
	select {
	case w.ch <- WebhookEvent{Event: event, Path: path, At: time.Now().UTC().Format(timeRFC3339)}:
	default:
		log.Printf("docs: webhook 队列满，丢弃 %s %s", event, path)
	}
}

// loop worker 主循环：出队 → 现读配置 → 投递（退避 3 次）。
func (w *WebhookDispatcher) loop() {
	defer w.wg.Done()
	for {
		select {
		case <-w.done:
			return
		case ev := <-w.ch:
			if url := w.config(); url != "" {
				w.deliver(url, ev)
			}
		}
	}
}

// deliver 签名投递：X-Ven-Signature = hex(HMAC-SHA256(secret, body))；退避 3 次。
func (w *WebhookDispatcher) deliver(url string, ev WebhookEvent) {
	body, err := json.Marshal(ev)
	if err != nil {
		return
	}
	secret := w.secret()
	backoffs := []time.Duration{0, 2 * time.Second, 8 * time.Second}
	for i, wait := range backoffs {
		select {
		case <-w.done:
			return
		case <-time.After(wait):
		}
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if secret != "" {
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write(body)
			req.Header.Set("X-Ven-Signature", hex.EncodeToString(mac.Sum(nil)))
		}
		resp, err := w.client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return
			}
			log.Printf("docs: webhook %s → %s 非 2xx（第 %d 次）: %d", ev.Event, url, i+1, resp.StatusCode)
		} else {
			log.Printf("docs: webhook %s → %s 失败（第 %d 次）: %v", ev.Event, url, i+1, err)
		}
	}
	log.Printf("docs: webhook %s %s 最终投递失败（已重试 %d 次）", ev.Event, url, len(backoffs))
}

func (w *WebhookDispatcher) config() string {
	url, _ := w.store.Get(SettingWebhookURL)
	return strings.TrimSpace(url)
}

func (w *WebhookDispatcher) secret() string {
	secret, _ := w.store.Get(SettingWebhookSecret)
	return strings.TrimSpace(secret)
}
