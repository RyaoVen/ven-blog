/** 动态页（ISR 静态页）：说说时间线，点击卡片弹窗看详情；数据变更由框架失效再生 + SSE 推送刷新 */

import { useState } from "react";
import type { PageAppProps } from "../app/pageApp";
import { formatDateTime } from "../lib/format";
import { Layout } from "../lib/layout";
import { Modal } from "../lib/modal";
import { v } from "../lib/theme";
import type { Moment, MomentsState } from "./types";

export default function MomentsPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { moments: [] }) as MomentsState;
    const [selected, setSelected] = useState<Moment | null>(null);

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
                        <li key={m.id}>
                            <button
                                type="button"
                                onClick={() => setSelected(m)}
                                className="ven-card ven-card-hover"
                                style={{
                                    display: "block",
                                    width: "100%",
                                    textAlign: "left",
                                    padding: "18px 22px",
                                    border: `1px solid ${v.border}`,
                                    background: "none",
                                    font: "inherit",
                                    cursor: "pointer",
                                }}
                            >
                                <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 12 }}>
                                    <MomentAvatar name={m.authorName} />
                                    <span style={{ fontWeight: 550 }}>{m.authorName}</span>
                                    <span className="ven-meta">{formatDateTime(m.createdAt)}</span>
                                </div>
                                <p
                                    style={{
                                        margin: 0,
                                        whiteSpace: "pre-wrap",
                                        overflow: "hidden",
                                        display: "-webkit-box",
                                        WebkitLineClamp: 3,
                                        WebkitBoxOrient: "vertical",
                                    }}
                                >
                                    {m.content}
                                </p>
                            </button>
                        </li>
                    ))}
                </ul>
            )}
            <Modal open={selected !== null} onClose={() => setSelected(null)}>
                {selected && (
                    <div>
                        <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 16 }}>
                            <MomentAvatar name={selected.authorName} size={36} />
                            <div>
                                <div style={{ fontWeight: 650 }}>{selected.authorName}</div>
                                <span className="ven-meta">{formatDateTime(selected.createdAt)}</span>
                            </div>
                        </div>
                        <p style={{ margin: 0, whiteSpace: "pre-wrap", lineHeight: 1.8 }}>{selected.content}</p>
                    </div>
                )}
            </Modal>
        </Layout>
    );
}

/** 动态作者字母头像（方角黑底） */
function MomentAvatar({ name, size = 28 }: { name: string; size?: number }) {
    return (
        <span
            style={{
                width: size,
                height: size,
                borderRadius: 3,
                background: v.text,
                display: "inline-flex",
                alignItems: "center",
                justifyContent: "center",
                fontSize: size * 0.46,
                fontWeight: 700,
                fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
                color: v.bg,
                flexShrink: 0,
            }}
        >
            {name.slice(0, 1).toUpperCase()}
        </span>
    );
}
