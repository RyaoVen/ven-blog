#!/usr/bin/env python3
"""make_report — 读取 A/B 两版 raw.json，生成 SVG 图表与 Markdown 汇总报告。

用法：
    python3 bench/make_report.py --a bench/results/A/raw.json --b bench/results/B/raw.json \
        --out bench/results
"""

import argparse
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from svgkit import Chart  # noqa: E402


def series_by(levels, key, mul=1.0):
    xs = [lv["concurrency"] for lv in levels]
    ys = [round(lv[key] * mul, 3) for lv in levels]
    return xs, ys


def latency_series(levels):
    xs = [lv["concurrency"] for lv in levels]
    return {
        "p50": (xs, [lv["p50"] for lv in levels]),
        "p95": (xs, [lv["p95"] for lv in levels]),
        "p99": (xs, [lv["p99"] for lv in levels]),
    }


def save(chart, out_dir, name):
    path = os.path.join(out_dir, name)
    chart.save(path)
    return name


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--a", required=True, help="修正前 raw.json")
    ap.add_argument("--b", required=True, help="修正后 raw.json")
    ap.add_argument("--out", default="bench/results")
    ap.add_argument("--label-a", default="修正前 (before)")
    ap.add_argument("--label-b", default="修正后 (after)")
    args = ap.parse_args()

    A = json.load(open(args.a, encoding="utf-8"))
    B = json.load(open(args.b, encoding="utf-8"))
    out = args.out
    os.makedirs(out, exist_ok=True)
    charts = []

    def max_of(key):
        return max(
            max((lv[key] for lv in A["levels"]), default=1),
            max((lv[key] for lv in B["levels"]), default=1),
        )

    # 1. 吞吐 RPS vs 并发
    c = Chart("吞吐对比：RPS vs 并发数", "并发连接数", "RPS (req/s)")
    xa, ya = series_by(A["levels"], "rps")
    xb, yb = series_by(B["levels"], "rps")
    c.line(
        [{"name": args.label_a, "points": list(zip(xa, ya))},
         {"name": args.label_b, "points": list(zip(xb, yb))}],
        xmax=max(max(xa), max(xb)), ymax=max(max(ya), max(yb)) * 1.15,
        xticks=len(xa),
    )
    charts.append(save(c, out, "1-rps-vs-concurrency.svg"))

    # 2. 延迟分位 vs 并发（p95/p99，A/B 各一组 → 4 系列）
    la, lb = latency_series(A["levels"]), latency_series(B["levels"])
    series = [
        {"name": f"{args.label_a} p95", "points": list(zip(*la["p95"]))},
        {"name": f"{args.label_b} p95", "points": list(zip(*lb["p95"]))},
        {"name": f"{args.label_a} p99", "points": list(zip(*la["p99"]))},
        {"name": f"{args.label_b} p99", "points": list(zip(*lb["p99"]))},
    ]
    ymax = max(max(y) for _, y in list(la.values()) + list(lb.values())) * 1.15 or 1
    c = Chart("延迟分位对比：p95/p99 vs 并发数", "并发连接数", "延迟 (s)")
    c.line(series, xmax=max(la["p50"][0]), ymax=ymax, xticks=len(la["p50"][0]))
    charts.append(save(c, out, "2-latency-percentiles.svg"))

    # 3. 平均延迟 vs 并发
    c = Chart("平均延迟对比", "并发连接数", "平均延迟 (s)")
    xa, ya = series_by(A["levels"], "latency_avg")
    xb, yb = series_by(B["levels"], "latency_avg")
    c.line(
        [{"name": args.label_a, "points": list(zip(xa, ya))},
         {"name": args.label_b, "points": list(zip(xb, yb))}],
        xmax=max(max(xa), max(xb)), ymax=max(max(ya), max(yb)) * 1.15,
        xticks=len(xa),
    )
    charts.append(save(c, out, "3-latency-avg.svg"))

    # 4. 错误率 vs 并发
    c = Chart("错误率对比", "并发连接数", "错误率")
    xa, ya = series_by(A["levels"], "error_rate", 100)
    xb, yb = series_by(B["levels"], "error_rate", 100)
    c.line(
        [{"name": args.label_a, "points": list(zip(xa, ya))},
         {"name": args.label_b, "points": list(zip(xb, yb))}],
        xmax=max(max(xa), max(xb)), ymax=100, xticks=len(xa), yticks=4,
    )
    charts.append(save(c, out, "4-error-rate.svg"))

    # 5. 资源利用率：Go/Node CPU 峰值 vs 并发
    def res_series(res, metric):
        out_s = {}
        for proc in ("go", "node"):
            arr = res["resource"]["series"][metric][proc]
            concs = [lv["concurrency"] for lv in res["levels"]]
            # 采样按时间均匀，按并发档均分切片取均值
            per = len(arr) // len(concs) if concs else 0
            out_s[proc] = (
                concs,
                [round(sum(arr[i * per:(i + 1) * per]) / per, 1) if per else 0 for i in range(len(concs))],
            )
        return out_s

    ra, rb = res_series(A, "cpu"), res_series(B, "cpu")
    series = [
        {"name": f"{args.label_a} Go", "points": list(zip(*ra["go"]))},
        {"name": f"{args.label_b} Go", "points": list(zip(*rb["go"]))},
        {"name": f"{args.label_a} Node", "points": list(zip(*ra["node"]))},
        {"name": f"{args.label_b} Node", "points": list(zip(*rb["node"]))},
    ]
    ymax = max(pt[1] for s in series for pt in s["points"]) * 1.15 or 1
    c = Chart("资源利用率：Go/Node CPU 均值 vs 并发", "并发连接数", "CPU (%)")
    c.line(series, xmax=max(ra["go"][0]), ymax=ymax, xticks=len(ra["go"][0]))
    charts.append(save(c, out, "5-cpu-vs-concurrency.svg"))

    # 6. 内存峰值条形图
    groups = ["Go 网关", "Node SSR", "MySQL"]
    values = [
        [A["resource"]["peak"]["go"]["rss"], B["resource"]["peak"]["go"]["rss"]],
        [A["resource"]["peak"]["node"]["rss"], B["resource"]["peak"]["node"]["rss"]],
        [A["resource"]["peak"]["mysqld"]["rss"], B["resource"]["peak"]["mysqld"]["rss"]],
    ]
    ymax = max(max(v) for v in values) * 1.15 or 1
    c = Chart("内存峰值对比 (RSS MB)", "进程", "RSS (MB)")
    c.bars(groups, [args.label_a, args.label_b], values, ymax)
    charts.append(save(c, out, "6-memory-peak.svg"))

    # 7. 缓存命中率 + API 体积对比
    groups = ["页面缓存命中率 (%)", "/api/posts 响应 (KB)"]
    def cum_rate(res):
        c = res.get("cache", {})
        return (c.get("cumulative_rate") or c.get("hit_rate") or 0) * 100
    a_cache, b_cache = cum_rate(A), cum_rate(B)
    a_after, b_after = A.get("cache", {}).get("after", {}), B.get("cache", {}).get("after", {})
    a_api = A.get("api", {}).get("full_bytes", 0) / 1024
    b_api = B.get("api", {}).get("full_bytes", 0) / 1024
    values = [[round(a_cache, 1), round(b_cache, 1)], [round(a_api, 0), round(b_api, 0)]]
    ymax = max(max(v) for v in values) * 1.2 or 1
    c = Chart("缓存命中率 与 API 响应体积", "指标", "数值")
    c.bars(groups, [args.label_a, args.label_b], values, ymax)
    charts.append(save(c, out, "7-cache-and-api-size.svg"))

    # 8. 汇总 Markdown
    def highest_load(levels):
        """最高承载：错误率 ≤1% 的最高并发档（且 RPS 不低于前档 60%）。"""
        best = None
        prev_rps = 0
        for lv in levels:
            if lv["error_rate"] <= 0.01 and lv["rps"] >= prev_rps * 0.6:
                best = lv
            prev_rps = lv["rps"]
        return best

    ha, hb = highest_load(A["levels"]), highest_load(B["levels"])

    def lv_row(name, lv):
        if not lv:
            return f"| {name} | — | — | — | — |"
        return (f"| {name} | {lv['concurrency']} | {lv['rps']} | "
                f"{lv['p95']}s | {lv['error_rate']:.2%} |")

    lines = [
        "# 基准测试对比报告",
        "",
        f"- 修正前：{A['label']}（{A['started_at']}）",
        f"- 修正后：{B['label']}（{B['started_at']}）",
        f"- 环境：同机（本机）同 MySQL 实例、同端口、同种子数据；每档 {A['levels'][0]['wall_seconds']}s",
        "",
        "## 最高承载（错误率 ≤1% 的最高并发档）",
        "",
        "| 版本 | 最高稳定并发 | RPS | p95 | 错误率 |",
        "|---|---|---|---|---|",
        lv_row(args.label_a, ha),
        lv_row(args.label_b, hb),
        "",
        "## 阶梯明细",
        "",
        "| 并发 | A RPS | B RPS | A p50 | B p50 | A p95 | B p95 | A p99 | B p99 | A 错误率 | B 错误率 |",
        "|---|---|---|---|---|---|---|---|---|---|---|",
    ]
    for la, lb in zip(A["levels"], B["levels"]):
        lines.append(
            f"| {la['concurrency']} | {la['rps']} | {lb['rps']} | {la['p50']} | {lb['p50']} "
            f"| {la['p95']} | {lb['p95']} | {la['p99']} | {lb['p99']} "
            f"| {la['error_rate']:.2%} | {lb['error_rate']:.2%} |"
        )
    lines += [
        "",
        "## 缓存与载荷",
        "",
        f"- 页面缓存命中率（累计口径）：修正前 {a_cache:.1f}%（hits={a_after.get('hits', 0):,}/miss={a_after.get('misses', 0):,}），"
        f"修正后 {b_cache:.1f}%（hits={b_after.get('hits', 0):,}/miss={b_after.get('misses', 0):,}）",
        f"- /api/posts 完整响应体积：修正前 {a_api:.0f}KB，修正后 {b_api:.0f}KB"
        + (f"（分页后 {B.get('api', {}).get('paged_bytes', 0) / 1024:.0f}KB）" if B.get("api", {}).get("paged_bytes", -1) > 0 else ""),
        "",
        "## 资源利用率（压测窗口均值/峰值）",
        "",
        "| 版本 | Go CPU 均值 | Node CPU 均值 | Go RSS 峰值 | Node RSS 峰值 | MySQL RSS 峰值 |",
        "|---|---|---|---|---|---|",
        f"| {args.label_a} | {A['resource']['avg_cpu']['go']}% | {A['resource']['avg_cpu']['node']}% "
        f"| {A['resource']['peak']['go']['rss']:.0f}MB | {A['resource']['peak']['node']['rss']:.0f}MB "
        f"| {A['resource']['peak']['mysqld']['rss']:.0f}MB |",
        f"| {args.label_b} | {B['resource']['avg_cpu']['go']}% | {B['resource']['avg_cpu']['node']}% "
        f"| {B['resource']['peak']['go']['rss']:.0f}MB | {B['resource']['peak']['node']['rss']:.0f}MB "
        f"| {B['resource']['peak']['mysqld']['rss']:.0f}MB |",
        "",
        "## 图表",
        "",
    ]
    for name in charts:
        lines.append(f"![{name}]({name})\n")

    report = os.path.join(out, "report.md")
    with open(report, "w", encoding="utf-8") as f:
        f.write("\n".join(lines))
    print(f"报告与 {len(charts)} 张 SVG 图表 → {out}")
    print(report.split("## 最高承载")[0])


if __name__ == "__main__":
    main()
