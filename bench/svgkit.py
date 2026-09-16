#!/usr/bin/env python3
"""svgkit — 零依赖 SVG 图表生成器（折线图 / 条形图）。

所有图表输出独立 <svg> 文件，可直接嵌入 Markdown 或浏览器查看。
配色固定：对比场景 A=steelblue，B=indianred，附加系列依 PALETTE 顺延。
"""

PALETTE = ["#4682b4", "#cd5c5c", "#2e8b57", "#daa520", "#7b68ee", "#556b2f"]


def _esc(s):
    return (str(s)).replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def _fmt(v):
    if v >= 1000:
        return f"{v:,.0f}"
    if v >= 1:
        return f"{v:.1f}".rstrip("0").rstrip(".")
    if v > 0:
        return f"{v:.3f}"
    return "0"


class Chart:
    """SVG 画布：坐标系统一为 900x520（含轴与图例）。"""

    W, H = 900, 520
    ML, MR, MT, MB = 90, 30, 50, 70  # 边距
    PW, PH = W - ML - MR, H - MT - MB  # 绘图区

    def __init__(self, title, xlabel, ylabel):
        self.title, self.xlabel, self.ylabel = title, xlabel, ylabel
        self.parts = []

    def _axes(self, xmax, ymax, xticks, yticks):
        p = self.parts
        p.append(
            f'<rect x="{self.ML}" y="{self.MT}" width="{self.PW}" height="{self.PH}" '
            f'fill="#fbfbfd" stroke="#c8c8d0"/>'
        )
        for i in range(yticks + 1):
            y = self.MT + self.PH * i / yticks
            val = ymax * i / yticks
            p.append(f'<line x1="{self.ML}" y1="{y:.1f}" x2="{self.ML + self.PW}" y2="{y:.1f}" stroke="#e2e2ea"/>')
            p.append(
                f'<text x="{self.ML - 8}" y="{y + 4:.1f}" text-anchor="end" font-size="12" fill="#555">{_fmt(val)}</text>'
            )
        for i, xv in enumerate(xticks):
            x = self.ML + self.PW * xv / xmax if xmax else self.ML
            p.append(f'<line x1="{x:.1f}" y1="{self.MT}" x2="{x:.1f}" y2="{self.MT + self.PH}" stroke="#e2e2ea"/>')
            p.append(
                f'<text x="{x:.1f}" y="{self.MT + self.PH + 18}" text-anchor="middle" font-size="12" fill="#555">{_fmt(xv)}</text>'
            )
        p.append(f'<text x="{self.W / 2}" y="{self.H - 12}" text-anchor="middle" font-size="14" fill="#333">{_esc(self.xlabel)}</text>')
        p.append(
            f'<text x="18" y="{self.H / 2}" text-anchor="middle" font-size="14" fill="#333" '
            f'transform="rotate(-90 18 {self.H / 2})">{_esc(self.ylabel)}</text>'
        )
        p.append(f'<text x="{self.W / 2}" y="26" text-anchor="middle" font-size="18" font-weight="bold" fill="#111">{_esc(self.title)}</text>')

    def line(self, series, xmax, ymax, xticks=6, yticks=5, markers=True):
        """series: [{name, points: [(x, y)]}]；xticks 可传列表或档位数"""
        if isinstance(xticks, int):
            xticks = [xmax * i / xticks for i in range(1, xticks + 1)]
        self._axes(xmax, ymax, xticks, yticks)
        for si, s in enumerate(series):
            color = PALETTE[si % len(PALETTE)]
            pts = " ".join(
                f"{self.ML + self.PW * x / xmax:.1f},{self.MT + self.PH - self.PH * min(y, ymax) / ymax:.1f}"
                for x, y in s["points"]
            )
            self.parts.append(
                f'<polyline points="{pts}" fill="none" stroke="{color}" stroke-width="2.5"/>'
            )
            if markers:
                for x, y in s["points"]:
                    cx = self.ML + self.PW * x / xmax
                    cy = self.MT + self.PH - self.PH * min(y, ymax) / ymax
                    self.parts.append(f'<circle cx="{cx:.1f}" cy="{cy:.1f}" r="4" fill="{color}"/>')
        self._legend([s["name"] for s in series])

    def bars(self, groups, series_names, values, ymax, yticks=5):
        """分组条形图：groups=[label]，values[group][si] = 数值。"""
        xticks_max = max((sum(v) for v in values), default=1) or 1
        self._axes(len(groups), ymax, list(range(1, len(groups) + 1)), yticks)
        gw = self.PW / len(groups)
        bw = gw / (len(series_names) + 1) * 0.8
        for gi, group in enumerate(groups):
            for si in range(len(series_names)):
                v = min(values[gi][si], ymax)
                x = self.ML + gi * gw + gw * 0.1 + si * (gw * 0.8 / len(series_names))
                h = self.PH * v / ymax
                color = PALETTE[si % len(PALETTE)]
                self.parts.append(
                    f'<rect x="{x:.1f}" y="{self.MT + self.PH - h:.1f}" width="{bw:.1f}" height="{h:.1f}" '
                    f'fill="{color}" stroke="#333" stroke-width="0.5"/>'
                )
                self.parts.append(
                    f'<text x="{x + bw / 2:.1f}" y="{self.MT + self.PH - h - 6:.1f}" text-anchor="middle" '
                    f'font-size="11" fill="#333">{_fmt(values[gi][si])}</text>'
                )
            self.parts.append(
                f'<text x="{self.ML + gi * gw + gw / 2:.1f}" y="{self.MT + self.PH + 18}" '
                f'text-anchor="middle" font-size="12" fill="#555">{_esc(group)}</text>'
            )
        self._legend(series_names)

    def _legend(self, names):
        lx = self.ML + 10
        for i, name in enumerate(names):
            color = PALETTE[i % len(PALETTE)]
            x = lx + i * 200
            self.parts.append(f'<rect x="{x}" y="{self.H - 34}" width="14" height="14" fill="{color}"/>')
            self.parts.append(f'<text x="{x + 20}" y="{self.H - 22}" font-size="13" fill="#333">{_esc(name)}</text>')

    def save(self, path):
        svg = (
            f'<svg xmlns="http://www.w3.org/2000/svg" width="{self.W}" height="{self.H}" '
            f'viewBox="0 0 {self.W} {self.H}" font-family="Helvetica,Arial,sans-serif">'
            f'<rect width="{self.W}" height="{self.H}" fill="#ffffff"/>'
            + "".join(self.parts)
            + "</svg>"
        )
        with open(path, "w", encoding="utf-8") as f:
            f.write(svg)
        return path
