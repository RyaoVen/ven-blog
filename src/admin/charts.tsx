/** 仪表盘 SVG 图表组件：折线图 / 增量卡 / 日历热力图 / 雷达图（手绘 SVG，无依赖） */

import { useMemo } from "react";
import { v } from "../lib/theme";

export interface DayCount {
    date: string;
    count: number;
}

export interface HeatDay {
    date: string;
    posts: number;
    chars: number;
}

export interface CategoryCount {
    category: string;
    count: number;
}

const mono = "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace";
const ACCENT = "#0d9488";

/** 增量卡三列网格：≤480px 纵向堆叠（组件自注入） */
const deltaCss = `
.ven-delta-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; }
@media (max-width: 480px) {
    .ven-delta-grid { grid-template-columns: 1fr; }
}
`;

/* ===== 折线图（近 7/30/365 日注册数） ===== */
export function LineChart({ data, height = 180 }: { data: DayCount[]; height?: number }) {
    const width = 720;
    const padL = 34;
    const padR = 10;
    const padT = 12;
    const padB = 22;

    const { points, max } = useMemo(() => {
        const maxValue = Math.max(1, ...data.map((d) => d.count));
        const innerW = width - padL - padR;
        const innerH = height - padT - padB;
        const pts = data.map((d, i) => {
            const x = padL + (data.length <= 1 ? innerW / 2 : (i / (data.length - 1)) * innerW);
            const y = padT + innerH - (d.count / maxValue) * innerH;
            return `${x.toFixed(1)},${y.toFixed(1)}`;
        });
        return { points: pts, max: maxValue };
    }, [data, height]);

    if (data.length === 0) {
        return null;
    }
    const first = data[0];
    const last = data[data.length - 1];
    const areaPoints = `${padL},${height - padB} ${points.join(" ")} ${width - padR},${height - padB}`;

    return (
        <svg viewBox={`0 0 ${width} ${height}`} style={{ display: "block", width: "100%", height: "auto" }} role="img" aria-label="用户增长折线图">
            <line x1={padL} y1={height - padB} x2={width - padR} y2={height - padB} stroke={v.borderStrong} strokeWidth="1" />
            <text x={4} y={padT + 8} fontSize="10" fill={v.textMuted} fontFamily={mono}>
                {max}
            </text>
            <text x={4} y={height - padB + 4} fontSize="10" fill={v.textMuted} fontFamily={mono}>
                0
            </text>
            <polygon points={areaPoints} fill={ACCENT} opacity="0.08" />
            <polyline points={points.join(" ")} fill="none" stroke={ACCENT} strokeWidth="1.5" strokeLinejoin="round" strokeLinecap="round" />
            <text x={padL} y={height - 6} fontSize="10" fill={v.textMuted} fontFamily={mono}>
                {first.date.slice(5)}
            </text>
            <text x={width - padR} y={height - 6} fontSize="10" fill={v.textMuted} fontFamily={mono} textAnchor="end">
                {last.date.slice(5)}
            </text>
        </svg>
    );
}

/* ===== 增量对比卡（较昨日/上周/上月） ===== */
export function DeltaCards({ deltas }: { deltas: { yesterday: number; week: number; month: number } }) {
    const items: [string, number][] = [
        ["较昨日", deltas.yesterday],
        ["较上周", deltas.week],
        ["较上月", deltas.month],
    ];
    return (
        <>
            <style>{deltaCss}</style>
            <div className="ven-delta-grid">
                {items.map(([label, n]) => (
                    <div key={label} className="ven-card" style={{ padding: "14px 18px" }}>
                        <div className="ven-meta" style={{ marginBottom: 6 }}>
                            {label}新增用户
                        </div>
                        <div style={{ fontFamily: mono, fontSize: 22, fontWeight: 700, color: n > 0 ? v.accent : v.textSecondary }}>
                            {n > 0 ? `+${n}` : n}
                        </div>
                    </div>
                ))}
            </div>
        </>
    );
}

/* ===== 日历热力图（文章+动态 篇数/字数） ===== */
const HEAT_LEVELS = ["var(--bg-subtle)", "rgba(13,148,136,0.25)", "rgba(13,148,136,0.5)", "rgba(13,148,136,0.75)", "#0d9488"];

function heatLevel(posts: number): number {
    if (posts <= 0) {
        return 0;
    }
    if (posts === 1) {
        return 1;
    }
    if (posts === 2) {
        return 2;
    }
    if (posts <= 4) {
        return 3;
    }
    return 4;
}

export function CalendarHeatmap({ data, weeks = 53 }: { data: HeatDay[]; weeks?: number }) {
    // 以最后一天为基准向前铺 weeks 周（列=周，行=周一到周日）
    const cells = useMemo(() => {
        const byDate = new Map(data.map((d) => [d.date, d]));
        const today = new Date();
        // 对齐到最近一个周日
        const end = new Date(today);
        end.setDate(end.getDate() - end.getDay());
        const grid: (HeatDay | null)[][] = [];
        for (let w = weeks - 1; w >= 0; w--) {
            const col: (HeatDay | null)[] = [];
            for (let d = 0; d < 7; d++) {
                const date = new Date(end);
                date.setDate(end.getDate() - w * 7 + d);
                if (date > today) {
                    col.push(null);
                    continue;
                }
                const key = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
                col.push(byDate.get(key) ?? { date: key, posts: 0, chars: 0 });
            }
            grid.push(col);
        }
        return grid;
    }, [data, weeks]);

    const cell = 11;
    const gap = 3;
    const step = cell + gap;

    return (
        <div style={{ overflowX: "auto" }}>
            <svg
                viewBox={`0 0 ${weeks * step + 30} ${7 * step + 4}`}
                style={{ display: "block", width: "100%", minWidth: 500, height: "auto" }}
                role="img"
                aria-label="发布日历热力图"
            >
                {["一", "三", "五", "日"].map((label, i) => (
                    <text key={label} x={0} y={[1, 3, 5, 6][i] * step + cell - 2} fontSize="9" fill={v.textMuted} fontFamily={mono}>
                        {label}
                    </text>
                ))}
                {cells.map((col, x) =>
                    col.map((day, y) =>
                        day ? (
                            <rect
                                key={`${x}-${y}`}
                                x={22 + x * step}
                                y={y * step}
                                width={cell}
                                height={cell}
                                rx="2"
                                fill={HEAT_LEVELS[heatLevel(day.posts)]}
                            >
                                <title>{`${day.date}：${day.posts} 篇 · ${day.chars} 字`}</title>
                            </rect>
                        ) : null,
                    ),
                )}
            </svg>
        </div>
    );
}

/* ===== 雷达图（分类发布数） ===== */
export function RadarChart({ data, size = 240 }: { data: CategoryCount[]; size?: number }) {
    if (data.length < 3) {
        return (
            <p style={{ color: v.textMuted, fontSize: 13 }}>
                分类不足 3 个，暂无雷达图（当前 {data.length} 个）。
            </p>
        );
    }
    const cx = size / 2;
    const cy = size / 2;
    const radius = size / 2 - 34;
    const max = Math.max(1, ...data.map((d) => d.count));
    const angleOf = (i: number) => -Math.PI / 2 + (i * 2 * Math.PI) / data.length;
    const pointOf = (i: number, r: number) => `${(cx + r * Math.cos(angleOf(i))).toFixed(1)},${(cy + r * Math.sin(angleOf(i))).toFixed(1)}`;
    const valuePoints = data.map((d, i) => pointOf(i, (d.count / max) * radius)).join(" ");

    return (
        <svg viewBox={`0 0 ${size} ${size}`} style={{ display: "block", width: "100%", maxWidth: size, height: "auto", margin: "0 auto" }} role="img" aria-label="分类发布雷达图">
            {[0.33, 0.66, 1].map((ratio) => (
                <polygon
                    key={ratio}
                    points={data.map((_, i) => pointOf(i, radius * ratio)).join(" ")}
                    fill="none"
                    stroke={v.border}
                    strokeWidth="1"
                />
            ))}
            {data.map((d, i) => {
                const labelR = radius + 18;
                const x = cx + labelR * Math.cos(angleOf(i));
                const y = cy + labelR * Math.sin(angleOf(i));
                return (
                    <g key={d.category}>
                        <line x1={cx} y1={cy} x2={cx + radius * Math.cos(angleOf(i))} y2={cy + radius * Math.sin(angleOf(i))} stroke={v.border} strokeWidth="1" />
                        <text x={x} y={y} fontSize="10" fill={v.textSecondary} fontFamily={mono} textAnchor="middle" dominantBaseline="middle">
                            {d.category} {d.count}
                        </text>
                    </g>
                );
            })}
            <polygon points={valuePoints} fill={ACCENT} opacity="0.15" stroke={ACCENT} strokeWidth="1.5" strokeLinejoin="round" />
            {data.map((d, i) => {
                const r = (d.count / max) * radius;
                return <circle key={i} cx={cx + r * Math.cos(angleOf(i))} cy={cy + r * Math.sin(angleOf(i))} r="2.5" fill={ACCENT} />;
            })}
        </svg>
    );
}

/* ===== 折线图范围切换 ===== */
export function RangeTabs({ range, onChange }: { range: string; onChange: (r: string) => void }) {
    const tabs = ["7", "30", "365"];
    return (
        <div style={{ display: "flex", gap: 14 }}>
            {tabs.map((t) => (
                <button
                    key={t}
                    type="button"
                    onClick={() => onChange(t)}
                    className="ven-meta"
                    style={{
                        border: "none",
                        background: "none",
                        padding: 0,
                        cursor: "pointer",
                        color: range === t ? v.accent : v.textMuted,
                        fontWeight: range === t ? 700 : 400,
                    }}
                >
                    近{t}日
                </button>
            ))}
        </div>
    );
}

export { useState };
