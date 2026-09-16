#!/usr/bin/env python3
"""sampler — 资源利用率采样（零依赖，macOS/Linux ps 统一解析）。

按进程名前缀采样 CPU% 与 RSS(MB)，输出时间序列：
    {"t": [0,1,2...], "go": {"cpu": [...], "rss": [...]}, "node": {...}, "mysqld": {...}}
"""

import subprocess
import threading
import time

TARGETS = {"go": "ven-blog-gateway", "node": "node", "mysqld": "mysqld"}


def _snapshot():
    """一次 ps 快照：返回 {target: {"cpu": x, "rss_mb": y}} 聚合值。"""
    out = {"go": 0.0, "node": 0.0, "mysqld": 0.0}
    rss = {"go": 0.0, "node": 0.0, "mysqld": 0.0}
    try:
        raw = subprocess.run(
            ["ps", "-axo", "pcpu,rss,comm"], capture_output=True, text=True, timeout=5
        ).stdout
    except Exception:
        return None
    for line in raw.splitlines()[1:]:
        parts = line.split(None, 2)
        if len(parts) != 3:
            continue
        try:
            cpu, rss_kb, comm = float(parts[0]), float(parts[1]), parts[2].strip()
        except ValueError:
            continue
        for target, prefix in TARGETS.items():
            # comm 可能带路径；匹配末段前缀（ven-blog-gateway / node / mysqld）
            name = comm.rsplit("/", 1)[-1]
            if name.startswith(prefix) and "grep" not in name:
                out[target] += cpu
                rss[target] += rss_kb / 1024.0
    return {"cpu": out, "rss": rss}


class Sampler:
    def __init__(self, interval=1.0):
        self.interval = interval
        self.stop_flag = threading.Event()
        self.thread = None
        self.series = {"t": [], "cpu": {k: [] for k in TARGETS}, "rss": {k: [] for k in TARGETS}}
        self._max = {k: {"cpu": 0.0, "rss": 0.0} for k in TARGETS}

    def start(self):
        def loop():
            t0 = time.time()
            while not self.stop_flag.is_set():
                snap = _snapshot()
                if snap:
                    t = round(time.time() - t0, 1)
                    self.series["t"].append(t)
                    for k in TARGETS:
                        self.series["cpu"][k].append(round(snap["cpu"][k], 1))
                        self.series["rss"][k].append(round(snap["rss"][k], 1))
                        self._max[k]["cpu"] = max(self._max[k]["cpu"], snap["cpu"][k])
                        self._max[k]["rss"] = max(self._max[k]["rss"], snap["rss"][k])
                self.stop_flag.wait(self.interval)

        self.thread = threading.Thread(target=loop, daemon=True)
        self.thread.start()

    def stop(self):
        self.stop_flag.set()
        if self.thread:
            self.thread.join(timeout=5)
        return {
            "series": self.series,
            "peak": self._max,
            "avg_cpu": {
                k: (round(sum(v) / len(v), 1) if v else 0.0) for k, v in self.series["cpu"].items()
            },
        }
