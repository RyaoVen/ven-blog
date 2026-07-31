/** 后台-数据面板：八项统计卡 + 最近评论 */

import type { PageAppProps } from "../app/pageApp";
import { formatDateTime } from "../lib/format";
import { v } from "../lib/theme";
import { AdminLayout } from "./adminLayout";
import type { AdminDashboardState } from "./types";

const mono = "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace";

const EMPTY: AdminDashboardState = {
    stats: { posts: 0, words: 0, comments: 0, likes: 0, favorites: 0, users: 0, moments: 0, subscribers: 0 },
    recentComments: [],
};

export default function AdminDashboardPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? EMPTY) as AdminDashboardState;
    const { stats } = state;
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
    return (
        <AdminLayout route={bootstrap.route}>
            <div
                style={{
                    display: "grid",
                    gridTemplateColumns: "repeat(auto-fill, minmax(140px, 1fr))",
                    gap: 14,
                    marginBottom: 36,
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
