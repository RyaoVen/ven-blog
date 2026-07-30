/** 动态页（ISR 静态页）：说说时间线；数据变更由框架失效再生 + SSE 推送刷新，页面零接入 */

import type { PageAppProps } from "../app/pageApp";
import { formatDateTime } from "../lib/format";
import { Layout } from "../lib/layout";
import { v } from "../lib/theme";

/** 一条动态（与 Go 侧 build/interfaces MomentView 的 JSON 同形） */
interface Moment {
    id: string;
    content: string;
    authorName: string;
    createdAt: string;
}

/** 动态页的 initialState */
interface MomentsState {
    moments: Moment[];
}

export default function MomentsPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { moments: [] }) as MomentsState;
    return (
        <Layout>
            <header style={{ marginBottom: 24 }}>
                <h1 style={{ fontSize: 28 }}>动态</h1>
                <p className="ven-meta" style={{ margin: 0 }}>
                    共 {state.moments.length} 条
                </p>
            </header>
            {state.moments.length === 0 ? (
                <p style={{ color: v.textSecondary }}>还没有动态。</p>
            ) : (
                <ul
                    style={{
                        listStyle: "none",
                        padding: 0,
                        margin: 0,
                        display: "flex",
                        flexDirection: "column",
                        gap: 16,
                    }}
                >
                    {state.moments.map((m) => (
                        <li key={m.id} className="ven-card" style={{ padding: "18px 22px" }}>
                            <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 12 }}>
                                <span
                                    style={{
                                        width: 28,
                                        height: 28,
                                        borderRadius: 2,
                                        background: v.text,
                                        display: "inline-flex",
                                        alignItems: "center",
                                        justifyContent: "center",
                                        fontSize: 13,
                                        fontWeight: 700,
                                        fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
                                        color: v.bg,
                                        flexShrink: 0,
                                    }}
                                >
                                    {m.authorName.slice(0, 1).toUpperCase()}
                                </span>
                                <span style={{ fontWeight: 550 }}>{m.authorName}</span>
                                <span className="ven-meta">{formatDateTime(m.createdAt)}</span>
                            </div>
                            <p style={{ margin: 0, whiteSpace: "pre-wrap" }}>{m.content}</p>
                        </li>
                    ))}
                </ul>
            )}
        </Layout>
    );
}
