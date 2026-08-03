// Node 熔断端到端测试：render 路径接入熔断器后的行为。
package httpserver

import (
	"errors"
	"strings"
	"testing"
	"time"

	"ven_hybird/internal/config"
	"ven_hybird/internal/pagepattern"
	"ven_hybird/internal/ssr"
)

// 连续失败达到阈值 → 熔断开启：快速失败 503 不再提交 Node；
// 半开间隔后放行一个试探请求，试探失败回到熔断。
func TestRender_CircuitBreakerFastFail(t *testing.T) {
	client := newChanClient()
	client.submitErr = errors.New("worker down")
	cfg := config.Config{
		NodeSubmitTimeout:    100 * time.Millisecond,
		RenderTimeout:        time.Second,
		InternalToken:        "secret",
		NodeCircuitThreshold: 1, // 一次失败即熔断（测状态机用最小区间）
		NodeCircuitHalfOpen:  60 * time.Millisecond,
	}
	s := New(cfg, client, ssr.NewPendingRegistry(8), ssr.CryptoHookIDGenerator{}, pagepattern.NewValidator(nil))

	// 第一次失败 → 502，熔断开启
	respCh := requestFallback(t, s, "/news/1")
	if status, _ := recvResponse(t, respCh); status != 502 {
		t.Fatalf("expected 502 first, got %d", status)
	}

	// 第二次：熔断快速失败 503，不再向 Node 提交任务
	respCh2 := requestFallback(t, s, "/news/1")
	status2, body2 := recvResponse(t, respCh2)
	if status2 != 503 || !strings.Contains(body2, "circuit") {
		t.Fatalf("expected 503 circuit open, got %d %q", status2, body2)
	}
	select {
	case task := <-client.tasks:
		t.Fatalf("circuit open should not submit task, got %+v", task)
	case <-time.After(150 * time.Millisecond):
	}

	// 半开间隔后：放行一个试探请求（提交失败 → 回到熔断）
	time.Sleep(70 * time.Millisecond)
	respCh3 := requestFallback(t, s, "/news/1")
	if status3, _ := recvResponse(t, respCh3); status3 != 502 {
		t.Fatalf("expected probe 502, got %d", status3)
	}

	// 试探失败后再次快速失败
	respCh4 := requestFallback(t, s, "/news/1")
	if status4, _ := recvResponse(t, respCh4); status4 != 503 {
		t.Fatalf("expected 503 after probe failure, got %d", status4)
	}
}
