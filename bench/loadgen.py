#!/usr/bin/env python3
"""loadgen — 零依赖 HTTP 负载发生器（线程池 + http.client keep-alive）。

用法（作为库）：
    gen = LoadGen("127.0.0.1", 8090, concurrency=16, duration=15, scenario=SCENARIO_MIX)
    result = gen.run()   # dict: rps/latency 分位/错误率/字节数/按端点分布
"""

import http.client
import random
import statistics
import threading
import time
from collections import Counter
from urllib.parse import urlparse

# 场景：端点权重 mix（模拟真实流量：首页/列表/详情/API/文档页）。
# 全部为公开 GET，命中 SSR/ISR/缓存/API 各链路。
SCENARIO_MIX = [
    ("/", 0.20),
    ("/posts", 0.15),
    ("/posts/1", 0.15),
    ("/posts/2", 0.10),
    ("/api/posts", 0.15),
    ("/api/posts?size=10&page=2", 0.05),
    ("/moments", 0.05),
    ("/site", 0.05),
    ("/search?q=go", 0.05),
]

# API 专项场景（对比分页/裁剪效果）
SCENARIO_API = [("/api/posts", 1.0)]


def _pick(scenario, rng):
    x = rng.random()
    acc = 0.0
    for path, w in scenario:
        acc += w
        if x <= acc:
            return path
    return scenario[-1][0]


class LoadGen:
    def __init__(self, host, port, concurrency, duration, scenario=None, timeout=10):
        self.host, self.port = host, port
        self.concurrency = concurrency
        self.duration = duration
        self.scenario = scenario or SCENARIO_MIX
        self.timeout = timeout
        self.stop_flag = threading.Event()
        self.lock = threading.Lock()
        self.latencies = []  # 秒
        self.status = Counter()
        self.bytes_total = 0
        self.endpoint_count = Counter()
        self.path_status = Counter()
        self.errors = Counter()

    def _request_once(self, conn, path):
        """单次请求；返回 (status, nbytes, conn)。连接失效返回 conn=None。"""
        conn.request("GET", path, headers={"Accept-Encoding": "identity", "Connection": "keep-alive"})
        resp = conn.getresponse()
        status = resp.status
        nbytes = len(resp.read())
        if resp.will_close:
            conn.close()
            conn = None
        return status, nbytes, conn

    def _worker(self, rng):
        # 每 worker 一条 keep-alive 连接；失效立即重建并重试一次
        #（消除压测器连接竞态噪声——服务端 keep-alive 关闭窗口期的失败不计入应用错误）。
        conn = None
        while not self.stop_flag.is_set():
            path = _pick(self.scenario, rng)
            t0 = time.perf_counter()
            status, nbytes = 0, 0
            try:
                if conn is None:
                    conn = http.client.HTTPConnection(self.host, self.port, timeout=self.timeout)
                status, nbytes, conn = self._request_once(conn, path)
            except Exception:
                try:
                    if conn:
                        conn.close()
                except Exception:
                    pass
                conn = None
                # 重试一次（全新连接）；仍失败才计连接错误。
                try:
                    conn = http.client.HTTPConnection(self.host, self.port, timeout=self.timeout)
                    status, nbytes, conn = self._request_once(conn, path)
                except Exception:
                    status = -1
                    try:
                        if conn:
                            conn.close()
                    except Exception:
                        pass
                    conn = None
            dt = time.perf_counter() - t0
            key = path.split("?")[0]
            with self.lock:
                if status == -1:
                    self.errors["conn"] += 1
                self.latencies.append(dt)
                self.status[status] += 1
                self.bytes_total += nbytes
                self.endpoint_count[key] += 1
                self.path_status[key + ":" + str(status)] += 1

    def _close_all(self):
        pass

    def run(self):
        rng = random.Random(time.time_ns() % (2**32))
        threads = []
        t_start = time.time()
        for _ in range(self.concurrency):
            t = threading.Thread(target=self._worker, args=(random.Random(rng.random()),), daemon=True)
            t.start()
            threads.append(t)
        # 固定时长后停机
        deadline = t_start + self.duration
        while time.time() < deadline:
            time.sleep(0.25)
        self.stop_flag.set()
        for t in threads:
            t.join(timeout=self.timeout + 2)
        wall = time.time() - t_start
        return self.summarize(wall)

    def summarize(self, wall):
        lat = sorted(self.latencies)
        n = len(lat)

        def pct(p):
            if not lat:
                return 0.0
            idx = min(n - 1, int(n * p))
            return lat[idx]

        total = sum(self.status.values()) + sum(self.errors.values())
        ok2xx = sum(v for k, v in self.status.items() if 200 <= k < 400)
        server_err = sum(v for k, v in self.status.items() if k >= 500) + self.status.get(-1, 0) + self.errors.get("conn", 0)
        return {
            "concurrency": self.concurrency,
            "wall_seconds": round(wall, 2),
            "requests": total,
            "rps": round(total / wall, 1) if wall > 0 else 0,
            "ok": ok2xx,
            "server_errors": server_err,
            "error_rate": round(server_err / total, 4) if total else 0,
            "latency_avg": round(statistics.mean(lat), 4) if lat else 0,
            "p50": round(pct(0.50), 4),
            "p95": round(pct(0.95), 4),
            "p99": round(pct(0.99), 4),
            "max": round(lat[-1], 4) if lat else 0,
            "bytes_total": self.bytes_total,
            "bytes_avg": round(self.bytes_total / total, 0) if total else 0,
            "status": dict(self.status),
            "endpoints": dict(self.endpoint_count),
            "path_status": dict(self.path_status),
        }
