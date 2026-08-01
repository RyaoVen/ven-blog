/** 后台-数据面板：统计卡 + 用户增长折线/增量 + 发布日历热力图 + 分类雷达图 + 最近评论 */

import { useState } from "react";
import type { PageAppProps } from "../app/pageApp";
import { formatDateTime } from "../lib/format";
import { v } from "../lib/theme";
import { AdminLayout } from "./adminLayout";
import { CalendarHeatmap, DeltaCards, LineChart, RadarChart, RangeTabs } from "./charts";
import type { AdminDashboardState } from "./types";

const mono = "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace";

const EMPTY: AdminDashboardState = {
    stats: { posts: 0, words: 0, comments: 0, likes: 0, favorites: 0, users: 0, moments: 0, subscribers: 0 },
    recentComments: [],
    userGrowth: { d7: [], d30: [], d365: [], deltas: { yesterday: 0, week: 0, month: 0 } },
    heatmap: [],
    categoryCounts: [],
};

export default function AdminDashboardPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? EMPTY) as AdminDashboardState;
    const { stats } = state;
    const [range, setRange] = useState("30");
    const cards: [string, number][] = [
        ["文章", stats.posts],
        ["总字数", stats.words],
        ["评论", stats.comments],
        ["点赞", stats.likes],
        ["收藏", stats.favorites],
        ["用户", stats.users],
        ["动态", stats.moments],
        ["订阅", stats.subscribers],
    ];
    const series = range === "7" ? state.userGrowth.d7 : range === "365" ? state.userGrowth.d365 : state.userGrowth.d30;

    return (
        <AdminLayout route={bootstrap.route}>
            <div
                style={{
                    display: "grid",
                    gridTemplateColumns: "repeat(auto-fill, minmax(140px, 1fr))",
                    gap: 14,
                    marginBottom: 32,
                }}
            >
                {cards.map(([label, value]) => (
                    <div key={label} className="ven-card" style={{ padding: "16px 18px" }}>
                        <div className="ven-meta" style={{ marginBottom: 6 }}>
                            {label}
                        </div>
                        <div style={{ fontFamily: mono, fontSize: 26, fontWeight: 700 }}>{value}</div>
                    </div>
                ))}
            </div>

            <section className="ven-card" style={{ padding: "20px 22px", marginBottom: 24 }}>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", marginBottom: 16 }}>
                    <h2 style={{ fontSize: 17, margin: 0 }}>用户增长</h2>
                    <RangeTabs range={range} onChange={setRange} />
                </div>
                <LineChart data={series} />
                <div style={{ marginTop: 18 }}>
                    <DeltaCards deltas={state.userGrowth.deltas} />
                </div>
            </section>

            <section className="ven-card" style={{ padding: "20px 22px", marginBottom: 24 }}>
                <h2 style={{ fontSize: 17, margin: "0 0 16px" }}>发布日历（近一年 · 文章 + 动态）</h2>
                <CalendarHeatmap data={state.heatmap} />
                <p className="ven-meta" style={{ margin: "10px 0 0" }}>
                    颜色越深发布越多（悬停看篇数与字数）
                </p>
            </section>

            <section className="ven-card" style={{ padding: "20px 22px", marginBottom: 24 }}>
                <h2 style={{ fontSize: 17, margin: "0 0 8px" }}>分类发布</h2>
                <RadarChart data={state.categoryCounts} />
            </section>

            <h2 style={{ fontSize: 18 }}>最近评论</h2>
            {state.recentComments.length === 0 ? (
                <p style={{ color: v.textMuted, fontSize: 14 }}>还没有评论。</p>
            ) : (
                <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                    {state.recentComments.map((c) => (
                        <li
                            key={c.id}
                            style={{ padding: "12px 0", borderBottom: `1px solid ${v.border}`, fontSize: 14 }}
                        >
                            <span style={{ fontWeight: 650 }}>{c.username}</span>
                            <span style={{ color: v.textMuted }}> 评论了 </span>
                            <a href={`/posts/${c.postId}`}>{c.postTitle || `#${c.postId}`}</a>
                            <span className="ven-meta" style={{ marginLeft: 10 }}>
                                {formatDateTime(c.createdAt)}
                            </span>
                            <p style={{ margin: "4px 0 0", color: v.textSecondary }}>{c.content}</p>
                        </li>
                    ))}
                </ul>
            )}
        </AdminLayout>
    );
}
