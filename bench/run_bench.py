#!/usr/bin/env python3
"""run_bench — 单版本基准测试主控。

前置：mysqld 已在 127.0.0.1:3307（root 无密码，库 ven_blog），种子数据已灌。
流程：启动被测（Node SSR + Go 网关）→ 预热 → 阶梯并发压测（资源采样、缓存统计并行）→ 停机 → raw.json

用法：
    python3 bench/run_bench.py --label A --node-dir /tmp/vb-a/frame/node \
        --go-bin /tmp/vb-a/go-bin --out bench/results
"""

import argparse
import json
import os
import signal
import subprocess
import sys
import time
import urllib.request

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from loadgen import LoadGen, SCENARIO_MIX, SCENARIO_API  # noqa: E402
from sampler import Sampler  # noqa: E402

PORT = 8090
HOST = "127.0.0.1"
ENV = {
    **os.environ,
    "BLOG_MYSQL_DSN": "root:@tcp(127.0.0.1:3307)/ven_blog?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
    "BLOG_AUTHOR_PASSWORD": "bench-author-2026",
    "VEN_INTERNAL_TOKEN": "bench-internal-token",
    "VEN_COOKIE_SECURE": "false",
    "VEN_LISTEN_ADDR": f":{PORT}",
    "VEN_RENDER_CALLBACK_URL": f"http://{HOST}:{PORT}/_internal/render-callback",
}


def http_get(path, timeout=10):
    req = urllib.request.Request(f"http://{HOST}:{PORT}{path}")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            body = r.read()
            return r.status, dict(r.headers), body
    except urllib.error.HTTPError as e:
        return e.code, dict(e.headers), e.read()
    except Exception as e:
        return -1, {}, str(e).encode()


def wait_up(deadline_s=60):
    t0 = time.time()
    while time.time() - t0 < deadline_s:
        code, _, _ = http_get("/healthz", timeout=3)
        if code == 200:
            return True
        time.sleep(1)
    return False


def cache_stats():
    code, _, body = http_get("/healthz", timeout=5)
    if code != 200:
        return {"hits": 0, "misses": 0}
    try:
        pc = json.loads(body)["pageCache"]
        return {"hits": int(pc["hits"]), "misses": int(pc["misses"])}
    except Exception:
        return {"hits": 0, "misses": 0}


def start_target(node_dir, go_bin, logs_dir):
    procs = []
    node_log = open(os.path.join(logs_dir, "node.log"), "w")
    go_log = open(os.path.join(logs_dir, "go.log"), "w")
    procs.append(subprocess.Popen(
        ["node", "dist/main.js"], cwd=node_dir, env=ENV,
        stdout=node_log, stderr=subprocess.STDOUT,
    ))
    time.sleep(3)
    # 固定进程名 ven-blog-gateway：资源采样器按名识别（与二进制实际版本无关）
    link = os.path.abspath(os.path.join(logs_dir, "ven-blog-gateway"))
    if os.path.lexists(link):
        os.remove(link)
    os.symlink(os.path.abspath(go_bin), link)
    procs.append(subprocess.Popen(
        [link], cwd=os.path.dirname(link), env=ENV,
        stdout=go_log, stderr=subprocess.STDOUT,
    ))
    if not wait_up(90):
        stop(procs)
        raise RuntimeError("被测服务未在 90s 内就绪（healthz 不通）")
    return procs


def stop(procs):
    for p in procs:
        try:
            p.send_signal(signal.SIGTERM)
        except Exception:
            pass
    time.sleep(2)
    for p in procs:
        try:
            p.kill()
        except Exception:
            pass


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--label", required=True)
    ap.add_argument("--node-dir", required=True, help="frame/node 目录（含 dist/main.js）")
    ap.add_argument("--go-bin", required=True)
    ap.add_argument("--out", default="bench/results")
    ap.add_argument("--levels", default="1,4,8,16,32,64,96")
    ap.add_argument("--duration", type=int, default=12, help="每档秒数")
    ap.add_argument("--warmup", type=int, default=10)
    args = ap.parse_args()

    out_dir = os.path.join(args.out, args.label)
    logs_dir = os.path.join(out_dir, "logs")
    os.makedirs(logs_dir, exist_ok=True)

    print(f"[{args.label}] 启动被测（node={args.node_dir} go={args.go_bin}）")
    procs = start_target(args.node_dir, args.go_bin, logs_dir)
    result = {"label": args.label, "started_at": time.strftime("%F %T"), "levels": [], "api": {}, "cache": {}}
    sampler = Sampler(interval=1.0)
    try:
        # 预热：低并发打满场景 + ISR 物化
        print(f"[{args.label}] 预热 {args.warmup}s（并发 4）")
        LoadGen(HOST, PORT, 4, args.warmup, SCENARIO_MIX).run()

        cache_before = cache_stats()
        sampler.start()

        levels = [int(x) for x in args.levels.split(",")]
        for conc in levels:
            print(f"[{args.label}] 阶梯并发 {conc}（{args.duration}s）")
            r = LoadGen(HOST, PORT, conc, args.duration, SCENARIO_MIX).run()
            result["levels"].append(r)
            print(f"    rps={r['rps']} p50={r['p50']}s p95={r['p95']}s err={r['error_rate']:.2%}")

        # API 专项：分页/裁剪效果（响应体积 + 简单负载）
        api_mix = LoadGen(HOST, PORT, 8, 8, SCENARIO_API).run()
        result["api"]["mix_rps"] = api_mix["rps"]
        result["api"]["bytes_avg"] = api_mix["bytes_avg"]
        code, headers, body = http_get("/api/posts")
        result["api"]["full_bytes"] = len(body)
        code2, _, body2 = http_get("/api/posts?size=10&page=1")
        result["api"]["paged_bytes"] = len(body2) if code2 == 200 else -1

        cache_after = cache_stats()
        dh, dm = cache_after["hits"] - cache_before["hits"], cache_after["misses"] - cache_before["misses"]
        total_after = cache_after["hits"] + cache_after["misses"]
        result["cache"] = {
            "before": cache_before, "after": cache_after,
            "delta_hits": dh, "delta_misses": dm,
            "hit_rate": round(dh / (dh + dm), 4) if (dh + dm) else 0,
            # 累计口径（防 Δ 口径受计数重置影响）：
            "cumulative_rate": round(cache_after["hits"] / total_after, 4) if total_after else 0,
        }
        result["resource"] = sampler.stop()
    finally:
        stop(procs)

    out_path = os.path.join(out_dir, "raw.json")
    with open(out_path, "w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=1)
    print(f"[{args.label}] 完成 → {out_path}")


if __name__ == "__main__":
    main()
