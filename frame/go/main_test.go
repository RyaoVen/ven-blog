// 启动编排测试：pattern 拉取失败时的高可用回退逻辑。
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ven_hybird/internal/config"
	"ven_hybird/internal/pagepattern"
)

// nodeUp 返回一个正常应答 /pages 的假 Node。
func nodeUp(patterns []string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"patterns": patterns})
	}))
}

// nodeDown 返回一个恒 503 的假 Node（模拟 Node 未就绪/宕机）。
func nodeDown() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
}

// Node 在线：直接拉取成功，不回退，并把 pattern 持久化到磁盘。
func TestLoadPatterns_NodeUpPersists(t *testing.T) {
	worker := nodeUp([]string{"/", "/posts/:id"})
	defer worker.Close()
	path := filepath.Join(t.TempDir(), "patterns.json")
	cfg := config.Config{
		NodeWorkerURL:     worker.URL,
		InternalToken:     "tok",
		NodeSubmitTimeout: 2 * time.Second,
		PatternsFile:      path,
	}

	validator, fallback, err := loadPatterns(cfg, 1)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if fallback {
		t.Fatal("expected fresh fetch, got fallback")
	}
	if err := validator.Validate("/posts/:id"); err != nil {
		t.Fatalf("unexpected validator: %v", err)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("patterns not persisted: %v", statErr)
	}
}

// Node 未就绪但有持久化副本：用持久化 pattern 启动（fallback=true），不失败。
func TestLoadPatterns_NodeDownUsesPersisted(t *testing.T) {
	worker := nodeDown()
	defer worker.Close()
	path := filepath.Join(t.TempDir(), "patterns.json")
	if err := pagepattern.Save(pagepattern.NewValidator([]string{"/", "/posts/:id"}), path); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	cfg := config.Config{
		NodeWorkerURL:     worker.URL,
		InternalToken:     "tok",
		NodeSubmitTimeout: 200 * time.Millisecond,
		PatternsFile:      path,
	}

	validator, fallback, err := loadPatterns(cfg, 1)
	if err != nil {
		t.Fatalf("expected persisted fallback, got err: %v", err)
	}
	if !fallback {
		t.Fatal("expected fallback=true")
	}
	if err := validator.Validate("/posts/:id"); err != nil {
		t.Fatalf("persisted validator unusable: %v", err)
	}
}

// 首启 Node 未就绪且无持久化副本：返回错误（main 据此失败退出，无法服务是合理的）。
func TestLoadPatterns_NodeDownNoPersisted(t *testing.T) {
	worker := nodeDown()
	defer worker.Close()
	cfg := config.Config{
		NodeWorkerURL:     worker.URL,
		InternalToken:     "tok",
		NodeSubmitTimeout: 200 * time.Millisecond,
		PatternsFile:      filepath.Join(t.TempDir(), "nope.json"),
	}

	if _, _, err := loadPatterns(cfg, 1); err == nil {
		t.Fatal("expected error when node down and no persisted patterns")
	}
}
